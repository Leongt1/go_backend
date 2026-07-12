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
	sslmode := dbCfg.SSLmode
	channelBinding := dbCfg.ChannelBinding

	if user == "" || host == "" || dbName == "" {
		return nil, fmt.Errorf("missing required database environment variables (DB_USER, DB_HOST, DB_NAME)")
	}
	if port == "" {
		port = "5432" // DB_PORT is optional; historic deploys never set it
	}

	dsn := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		user,
		password,
		host,
		port,
		dbName,
		sslmode,
	)
	// pgx forwards channel_binding to the server as a startup parameter, which
	// vanilla Postgres rejects; Neon's proxy accepts it. Only send it when it
	// is actually requested (anything other than "disable").
	if channelBinding != "" && channelBinding != "disable" {
		dsn += "&channel_binding=" + channelBinding
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parseerr: %w", err)
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
