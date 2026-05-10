package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
)

type ModelRoute struct {
	ClientModelID     string
	ChannelID         uint64
	UpstreamModelID   string
	Enabled           bool
}

type Channel struct {
	ID             uint64
	Name           string
	ChannelType    string
	BaseURL        string
	APIKeyCipher   string
	Status         string
}

func (s *Store) GetModelRoute(ctx context.Context, clientModel string) (*ModelRoute, error) {
	var r ModelRoute
	var en int
	err := s.DB.QueryRowContext(ctx,
		`SELECT client_model_id, channel_id, upstream_model_id, enabled FROM model_routes WHERE client_model_id = ?`,
		clientModel,
	).Scan(&r.ClientModelID, &r.ChannelID, &r.UpstreamModelID, &en)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.Enabled = en == 1
	return &r, nil
}

func (s *Store) GetChannel(ctx context.Context, id uint64) (*Channel, error) {
	var c Channel
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, name, channel_type, base_url, api_key_cipher, status FROM channels WHERE id = ?`,
		id,
	).Scan(&c.ID, &c.Name, &c.ChannelType, &c.BaseURL, &c.APIKeyCipher, &c.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &c, err
}

// DecodeChannelAPIKey extracts upstream API key from stored cipher (dev: plain:base64).
func DecodeChannelAPIKey(cipher string) (string, error) {
	if strings.HasPrefix(cipher, "plain:") {
		raw := cipher[len("plain"):]
		if strings.HasPrefix(raw, ":") {
			raw = raw[1:]
		}
		b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return "", errors.New("unsupported cipher (configure plain:base64 or extend KMS)")
}

func (s *Store) ListEnabledModels(ctx context.Context) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT model_id FROM platform_models WHERE enabled = 1 ORDER BY model_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
