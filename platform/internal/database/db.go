package database

import (
	"context"
	"fmt"
	"net/url"
	"time"

	// Initial Migration
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the Postgres database connection pool
type DB struct {
	Pool *pgxpool.Pool
}

// Config holds database configuration
type Config struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	Name           string `mapstructure:"name"`
	MigrationsPath string `mapstructure:"migrationspath"`
}

// NewDB creates a new database connection and runs migrations
func NewDB(cfg Config) (*DB, error) {
	// URL-encode password to handle special characters (/, +, =, etc.)
	encodedPassword := url.QueryEscape(cfg.Password)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User, encodedPassword, cfg.Host, cfg.Port, cfg.Name)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run Migrations
	if cfg.MigrationsPath != "" {
		m, err := migrate.New(
			"file://"+cfg.MigrationsPath,
			dsn,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create migration instance: %w", err)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return &DB{Pool: pool}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	db.Pool.Close()
}
