package store

import (
	"context"
	"database/sql"
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
