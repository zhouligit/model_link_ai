package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type AdminUserRow struct {
	ID          uint64
	Email       string
	DisplayName string
	Role        string
	Status      string
	CreatedAt   time.Time
}

func (s *Store) AdminListUsers(ctx context.Context, limit int) ([]AdminUserRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, email, COALESCE(display_name,''), role, status, created_at FROM users ORDER BY id DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminUserRow
	for rows.Next() {
		var r AdminUserRow
		var email sql.NullString
		var dn string
		if err := rows.Scan(&r.ID, &email, &dn, &r.Role, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			r.Email = email.String
		}
		r.DisplayName = dn
		out = append(out, r)
	}
	return out, rows.Err()
}

// AdminUserDetail is safe fields for admin console (no password).
type AdminUserDetail struct {
	ID          uint64
	Email       sql.NullString
	Phone       sql.NullString
	DisplayName sql.NullString
	AvatarURL   sql.NullString
	Role        string
	Status      string
	LastLoginAt sql.NullTime
	CreatedAt   time.Time
}

func (s *Store) AdminGetUserDetail(ctx context.Context, userID uint64) (*AdminUserDetail, error) {
	var d AdminUserDetail
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, email, phone, display_name, avatar_url, role, status, last_login_at, created_at
		 FROM users WHERE id = ?`,
		userID,
	).Scan(&d.ID, &d.Email, &d.Phone, &d.DisplayName, &d.AvatarURL, &d.Role, &d.Status, &d.LastLoginAt, &d.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// AdminUserWallet is personal wallet row for a user (owner_type=user).
type AdminUserWallet struct {
	BalanceCents int64  `json:"balance_cents"`
	Currency     string `json:"currency"`
	Status       string `json:"status"`
}

func (s *Store) AdminGetUserWallet(ctx context.Context, userID uint64) (*AdminUserWallet, error) {
	var w AdminUserWallet
	err := s.DB.QueryRowContext(ctx,
		`SELECT balance_cents, currency, status FROM wallet_accounts WHERE owner_type = 'user' AND owner_id = ?`,
		userID,
	).Scan(&w.BalanceCents, &w.Currency, &w.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// APIKeyAdminRow is api key metadata for admin (prefix only, never secret).
type APIKeyAdminRow struct {
	ID          uint64
	OrgID       *uint64
	Scope       string
	Name        string
	KeyPrefix   string
	Status      string
	LastUsedAt  sql.NullTime
	CreatedAt   time.Time
}

func (s *Store) AdminListAPIKeysForUser(ctx context.Context, userID uint64) ([]APIKeyAdminRow, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, org_id, scope, name, key_prefix, status, last_used_at, created_at
		 FROM api_keys WHERE user_id = ? ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKeyAdminRow
	for rows.Next() {
		var r APIKeyAdminRow
		var org sql.NullInt64
		if err := rows.Scan(&r.ID, &org, &r.Scope, &r.Name, &r.KeyPrefix, &r.Status, &r.LastUsedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if org.Valid {
			v := uint64(org.Int64)
			r.OrgID = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
