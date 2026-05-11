package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

type APIKeyRow struct {
	ID         uint64
	UserID     uint64
	OrgID      *uint64
	Scope      string
	Name       string
	KeyPrefix  string
	KeyHash    string
	Status     string
	LastUsedAt *string
	CreatedAt  string
}

func hashAPIKey(full string) string {
	h := sha256.Sum256([]byte(full))
	return hex.EncodeToString(h[:])
}

// GenerateAPIKey returns fullSecret (once), prefix for display, hash for storage（始终 mk_live_，无测试前缀分支）。
func GenerateAPIKey() (full string, prefix string, hash string, err error) {
	var b [24]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", "", err
	}
	const pref = "mk_live_"
	suffix := hex.EncodeToString(b[:])
	full = pref + suffix
	prefix = pref + suffix[:8]
	hash = hashAPIKey(full)
	return full, prefix, hash, nil
}

func (s *Store) InsertAPIKey(ctx context.Context, userID uint64, orgID *uint64, scope, name, keyPrefix, keyHash string) (uint64, error) {
	var oid interface{}
	if orgID != nil {
		oid = *orgID
	}
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO api_keys (user_id, org_id, scope, name, key_prefix, key_hash, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'active')`,
		userID, oid, scope, name, keyPrefix, keyHash,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func (s *Store) ResolveAPIKey(ctx context.Context, fullKey string) (*APIKeyRow, error) {
	h := hashAPIKey(strings.TrimSpace(fullKey))
	var row APIKeyRow
	var org sql.NullInt64
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, org_id, scope, name, key_prefix, key_hash, status FROM api_keys WHERE key_hash = ? AND status = 'active'`,
		h,
	).Scan(&row.ID, &row.UserID, &org, &row.Scope, &row.Name, &row.KeyPrefix, &row.KeyHash, &row.Status)
	if err != nil {
		return nil, err
	}
	if org.Valid {
		v := uint64(org.Int64)
		row.OrgID = &v
	}
	return &row, nil
}

func (s *Store) TouchAPIKeyUsed(ctx context.Context, id uint64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP(3) WHERE id = ?`, id)
	return err
}

func (s *Store) ListAPIKeys(ctx context.Context, userID uint64) ([]APIKeyRow, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, user_id, org_id, scope, name, key_prefix, status FROM api_keys WHERE user_id = ? ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKeyRow
	for rows.Next() {
		var row APIKeyRow
		var org sql.NullInt64
		if err := rows.Scan(&row.ID, &row.UserID, &org, &row.Scope, &row.Name, &row.KeyPrefix, &row.Status); err != nil {
			return nil, err
		}
		if org.Valid {
			v := uint64(org.Int64)
			row.OrgID = &v
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) DisableAPIKey(ctx context.Context, userID, keyID uint64) error {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE api_keys SET status = 'disabled' WHERE id = ? AND user_id = ?`,
		keyID, userID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("not found")
	}
	return nil
}
