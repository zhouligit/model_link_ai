package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type User struct {
	ID           uint64
	Email        string
	Phone        sql.NullString
	DisplayName  sql.NullString
	Role         string
	Status       string
	PasswordHash string
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName, role string) (uint64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (email, password_hash, display_name, role, status) VALUES (?, ?, ?, ?, 'active')`,
		nullIfEmpty(email), passwordHash, nullString(displayName), role,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint64(id), err
}

func nullIfEmpty(email string) interface{} {
	if email == "" {
		return nil
	}
	return email
}

func nullString(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	var u User
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, email, phone, display_name, role, status, password_hash FROM users WHERE email = ?`,
		email,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.DisplayName, &u.Role, &u.Status, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) GetUserByID(ctx context.Context, id uint64) (*User, error) {
	var u User
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, email, phone, display_name, role, status, password_hash FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Email, &u.Phone, &u.DisplayName, &u.Role, &u.Status, &u.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &u, err
}

func (s *Store) TouchLogin(ctx context.Context, userID uint64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now(), userID)
	return err
}

func (s *Store) InsertRefreshToken(ctx context.Context, userID uint64, tokenHash string, expiresAt time.Time, device string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO user_refresh_tokens (user_id, token_hash, expires_at, device_info) VALUES (?, ?, ?, ?)`,
		userID, tokenHash, expiresAt, nullString(device),
	)
	return err
}

func (s *Store) FindRefreshToken(ctx context.Context, tokenHash string) (userID uint64, expiresAt time.Time, revokedAt sql.NullTime, err error) {
	err = s.DB.QueryRowContext(ctx,
		`SELECT user_id, expires_at, revoked_at FROM user_refresh_tokens WHERE token_hash = ?`,
		tokenHash,
	).Scan(&userID, &expiresAt, &revokedAt)
	return
}

func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE user_refresh_tokens SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		time.Now(), tokenHash,
	)
	return err
}

func (s *Store) RevokeAllRefreshTokens(ctx context.Context, userID uint64) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE user_refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		time.Now(), userID,
	)
	return err
}

func (s *Store) UpdateProfile(ctx context.Context, userID uint64, displayName, avatarURL string) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE users SET display_name = COALESCE(?, display_name), avatar_url = COALESCE(?, avatar_url) WHERE id = ?`,
		nullString(displayName), nullString(avatarURL), userID,
	)
	return err
}
