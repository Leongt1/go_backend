package db

import (
	"backend-go/internal/platform/config"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, dbCfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	user := dbCfg.User
	password := dbCfg.Password
	host := dbCfg.Host
	port := dbCfg.Port
	dbName := dbCfg.Name

	if user == "" || host == "" || dbName == "" {
		return nil, fmt.Errorf("missing required database environment variables (DB_USER, DB_HOST, DB_NAME)")
	}

	dns := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		host,
		port,
		dbName,
	)

	cfg, err := pgxpool.ParseConfig(dns)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}
