package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres wraps the pgx connection pool
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres creates a new Postgres connection pool and initializes pgvector
func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 25
	config.MinConns = 5

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	slog.Info("database connection established", "max_conns", config.MaxConns)

	// Initialize pgvector extension (idempotent)
	if err := initPgVector(ctx, pool); err != nil {
		slog.Warn("pgvector initialization failed (may already exist)", "error", err)
	}

	return &Postgres{Pool: pool}, nil
}

// initPgVector creates the vector extension if it doesn't exist
func initPgVector(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("create vector extension: %w", err)
	}
	slog.Info("pgvector extension initialized")
	return nil
}

// Close closes the database connection pool
func (p *Postgres) Close() {
	p.Pool.Close()
	slog.Info("database connection pool closed")
}
