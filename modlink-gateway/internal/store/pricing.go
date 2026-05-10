package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) CurrentPricing(ctx context.Context, modelID string) (inputPer1kCents, outputPer1kCents int64, err error) {
	now := time.Now()
	err = s.DB.QueryRowContext(ctx,
		`SELECT input_per_1k_cents, output_per_1k_cents FROM pricing_models
		 WHERE model_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)
		 ORDER BY effective_from DESC LIMIT 1`,
		modelID, now, now,
	).Scan(&inputPer1kCents, &outputPer1kCents)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, 3, nil
	}
	return inputPer1kCents, outputPer1kCents, err
}

func EstimateCostCents(inputTok, outputTok int, inPer1k, outPer1k int64) int64 {
	inCost := (int64(inputTok) * inPer1k + 999) / 1000
	outCost := (int64(outputTok) * outPer1k + 999) / 1000
	return inCost + outCost
}
