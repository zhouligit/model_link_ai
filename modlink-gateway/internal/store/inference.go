package store

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) InsertInferenceLog(ctx context.Context, requestID string, apiKeyID, userID uint64, orgID *uint64, model string, channelID *uint64,
	inputTok, outputTok int, costCents int64, billingType string, httpStatus int, upstreamStatus string, latencyMs int, promptStored bool,
) error {
	var oid interface{}
	if orgID != nil {
		oid = *orgID
	}
	var ch interface{}
	if channelID != nil {
		ch = *channelID
	}
	ps := 0
	if promptStored {
		ps = 1
	}
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO inference_logs (request_id, api_key_id, user_id, org_id, model, channel_id, input_tokens, output_tokens, cost_cents, billing_type, http_status, upstream_status, latency_ms, prompt_stored)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE input_tokens=VALUES(input_tokens), output_tokens=VALUES(output_tokens), cost_cents=VALUES(cost_cents)`,
		requestID, apiKeyID, userID, oid, model, ch, inputTok, outputTok, costCents, billingType, httpStatus, upstreamStatus, latencyMs, ps,
	)
	return err
}

// UsageSummary returns aggregate stats from wallet debits (MVP).
func (s *Store) UsageSummary(ctx context.Context, walletID uint64, days int) (calls int64, costCents int64, err error) {
	since := time.Now().AddDate(0, 0, -days)
	err = s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(amount_cents), 0) FROM wallet_transactions
		 WHERE wallet_account_id = ? AND direction = 'debit' AND biz_type = 'inference' AND created_at >= ?`,
		walletID, since,
	).Scan(&calls, &costCents)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return calls, costCents, err
}
