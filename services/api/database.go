package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func InitDB() (*pgxpool.Pool, error) {
	// DATABASE_URL (single connection string) is preferred — used by Fly.io, Railway,
	// Render, and other managed platforms. Fall back to individual DB_* vars for
	// local Docker Compose.
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASSWORD")
		name := os.Getenv("DB_NAME")
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "5432"
		}
		// Use sslmode=disable only for localhost; require TLS for any remote host.
		sslMode := "disable"
		if host != "localhost" && host != "127.0.0.1" && !strings.HasPrefix(host, "postgres") {
			sslMode = "require"
		}
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			url.QueryEscape(user), url.QueryEscape(pass), host, port, name, sslMode)
	} else {
		// If DATABASE_URL is provided but has no sslmode and the host is not localhost,
		// inject sslmode=require so managed cloud databases use TLS automatically.
		connStr = ensureSSLMode(connStr)
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	db = pool
	return pool, nil
}

// ensureSSLMode adds sslmode=require to DATABASE_URL when the host is not localhost
// and the URL does not already specify sslmode. This ensures managed cloud databases
// (Render, Fly.io, Neon, Supabase) always use TLS without requiring the caller to
// remember to include the parameter.
func ensureSSLMode(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // unable to parse — return unchanged, let pgx report the error
	}
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "postgres") {
		return rawURL // local / Docker Compose — sslmode=disable is fine
	}
	q := u.Query()
	if q.Get("sslmode") == "" {
		q.Set("sslmode", "require")
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func GetDB() *pgxpool.Pool {
	return db
}
