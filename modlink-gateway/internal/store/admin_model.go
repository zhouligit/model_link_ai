package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrDuplicateModel  = errors.New("duplicate model_id or route")
)

func mysqlDuplicate(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// CreatePlatformModelBundle inserts platform_models, model_routes, pricing_models in one transaction.
func (s *Store) CreatePlatformModelBundle(ctx context.Context, modelID, displayName, upstreamModelID string, channelID uint64, inputPer1k, outputPer1k int64, enabled bool) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("model_id required")
	}
	if displayName == "" {
		displayName = modelID
	}
	upstreamModelID = strings.TrimSpace(upstreamModelID)
	if upstreamModelID == "" {
		upstreamModelID = modelID
	}
	if channelID == 0 {
		channelID = 1
	}
	if inputPer1k < 0 || outputPer1k < 0 {
		return fmt.Errorf("pricing must be non-negative")
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var ch int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM channels WHERE id = ? LIMIT 1`, channelID).Scan(&ch)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrChannelNotFound
	}
	if err != nil {
		return err
	}

	en := 0
	if enabled {
		en = 1
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO platform_models (model_id, display_name, enabled, default_channel_id) VALUES (?, ?, ?, ?)`,
		modelID, displayName, en, channelID,
	)
	if err != nil {
		if mysqlDuplicate(err) {
			return ErrDuplicateModel
		}
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO model_routes (client_model_id, channel_id, upstream_model_id, enabled) VALUES (?, ?, ?, 1)`,
		modelID, channelID, upstreamModelID,
	)
	if err != nil {
		if mysqlDuplicate(err) {
			return ErrDuplicateModel
		}
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO pricing_models (model_id, input_per_1k_cents, output_per_1k_cents) VALUES (?, ?, ?)`,
		modelID, inputPer1k, outputPer1k,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
