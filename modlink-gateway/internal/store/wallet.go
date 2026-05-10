package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type WalletAccount struct {
	ID            uint64
	OwnerType     string
	OwnerID       uint64
	BalanceCents  int64
	Currency      string
	Status        string
	Version       uint32
}

func (s *Store) EnsureWallet(ctx context.Context, ownerType string, ownerID uint64) (*WalletAccount, error) {
	var wa WalletAccount
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, owner_type, owner_id, balance_cents, currency, status, version
		 FROM wallet_accounts WHERE owner_type = ? AND owner_id = ?`,
		ownerType, ownerID,
	).Scan(&wa.ID, &wa.OwnerType, &wa.OwnerID, &wa.BalanceCents, &wa.Currency, &wa.Status, &wa.Version)
	if errors.Is(err, sql.ErrNoRows) {
		res, err := s.DB.ExecContext(ctx,
			`INSERT INTO wallet_accounts (owner_type, owner_id, balance_cents, currency, status, version)
			 VALUES (?, ?, 0, 'CNY', 'active', 0)`,
			ownerType, ownerID,
		)
		if err != nil {
			return nil, err
		}
		id, _ := res.LastInsertId()
		wa = WalletAccount{ID: uint64(id), OwnerType: ownerType, OwnerID: ownerID, BalanceCents: 0, Currency: "CNY", Status: "active", Version: 0}
		return &wa, nil
	}
	return &wa, err
}

// Credit adds funds (recharge). Idempotent by external ref in meta if needed at caller.
func (s *Store) Credit(ctx context.Context, walletID uint64, amountCents int64, biz string, refType string, refID uint64, remark string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var bal int64
	var ver uint32
	err = tx.QueryRowContext(ctx,
		`SELECT balance_cents, version FROM wallet_accounts WHERE id = ? FOR UPDATE`,
		walletID,
	).Scan(&bal, &ver)
	if err != nil {
		return err
	}
	newBal := bal + amountCents
	res, err := tx.ExecContext(ctx,
		`UPDATE wallet_accounts SET balance_cents = ?, version = version + 1 WHERE id = ? AND version = ?`,
		newBal, walletID, ver,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("wallet concurrent update")
	}
	meta, _ := json.Marshal(map[string]any{"ref_type": refType, "ref_id": refID})
	_, err = tx.ExecContext(ctx,
		`INSERT INTO wallet_transactions (wallet_account_id, direction, amount_cents, balance_after, biz_type, ref_type, ref_id, remark, meta)
		 VALUES (?, 'credit', ?, ?, ?, ?, ?, ?, ?)`,
		walletID, amountCents, newBal, biz, refType, refID, remark, meta,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DebitInference deducts for inference; requestID for idempotency.
func (s *Store) DebitInference(ctx context.Context, walletID uint64, amountCents int64, requestID string, billingType string, model string) error {
	if amountCents <= 0 {
		return nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var existing int
	_ = tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE request_id = ? AND biz_type = 'inference'`,
		requestID,
	).Scan(&existing)
	if existing > 0 {
		return tx.Commit()
	}

	var bal int64
	var ver uint32
	err = tx.QueryRowContext(ctx,
		`SELECT balance_cents, version FROM wallet_accounts WHERE id = ? FOR UPDATE`,
		walletID,
	).Scan(&bal, &ver)
	if err != nil {
		return err
	}
	if bal < amountCents {
		return fmt.Errorf("insufficient balance")
	}
	newBal := bal - amountCents
	res, err := tx.ExecContext(ctx,
		`UPDATE wallet_accounts SET balance_cents = ?, version = version + 1 WHERE id = ? AND version = ?`,
		newBal, walletID, ver,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("wallet concurrent update")
	}
	meta, _ := json.Marshal(map[string]any{"model": model})
	_, err = tx.ExecContext(ctx,
		`INSERT INTO wallet_transactions (wallet_account_id, direction, amount_cents, balance_after, biz_type, ref_type, ref_id, request_id, billing_type, remark, meta)
		 VALUES (?, 'debit', ?, ?, 'inference', 'inference_log', NULL, ?, ?, ?, ?)`,
		walletID, amountCents, newBal, requestID, billingType, model, meta,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) WalletForAPIKeyScope(ctx context.Context, scope string, userID uint64, orgID sql.NullInt64) (*WalletAccount, error) {
	switch scope {
	case "personal":
		return s.EnsureWallet(ctx, "user", userID)
	case "org":
		if !orgID.Valid {
			return nil, fmt.Errorf("org scope without org id")
		}
		return s.EnsureWallet(ctx, "org", uint64(orgID.Int64))
	default:
		return nil, fmt.Errorf("unknown scope")
	}
}

func (s *Store) SumSpendCentsSince(ctx context.Context, walletID uint64, since time.Time) (int64, error) {
	var sum sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount_cents), 0) FROM wallet_transactions
		 WHERE wallet_account_id = ? AND direction = 'debit' AND created_at >= ?`,
		walletID, since,
	).Scan(&sum)
	if err != nil {
		return 0, err
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Int64, nil
}
