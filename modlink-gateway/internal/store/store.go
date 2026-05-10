package store

import (
	"context"
	"database/sql"
)

type Store struct {
	DB *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{DB: db}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.DB.PingContext(ctx)
}
