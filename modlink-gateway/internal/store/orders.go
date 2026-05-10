package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Order struct {
	ID           uint64
	UserID       uint64
	OrgID        *uint64
	AmountCents  int64
	Channel      string
	Status       string
	CreatedAt    time.Time
}

func (s *Store) CreateOrder(ctx context.Context, userID uint64, orgID *uint64, amountCents int64, channel string) (uint64, error) {
	var oid interface{}
	if orgID != nil {
		oid = *orgID
	}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO orders (user_id, org_id, order_type, amount_cents, currency, channel, status)
		 VALUES (?, ?, 'recharge', ?, 'CNY', ?, 'pending')`,
		userID, oid, amountCents, channel,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) GetOrder(ctx context.Context, id uint64) (*Order, error) {
	var o Order
	var org sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, org_id, amount_cents, channel, status, created_at FROM orders WHERE id = ?`,
		id,
	).Scan(&o.ID, &o.UserID, &org, &o.AmountCents, &o.Channel, &o.Status, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if org.Valid {
		v := uint64(org.Int64)
		o.OrgID = &v
	}
	return &o, err
}

func (s *Store) MarkOrderPaid(ctx context.Context, orderID uint64, providerTradeNo string) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE orders SET status = 'paid', provider_trade_no = ?, paid_at = CURRENT_TIMESTAMP(3) WHERE id = ? AND status = 'pending'`,
		providerTradeNo, orderID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return errors.New("order not pending")
	}
	return nil
}
