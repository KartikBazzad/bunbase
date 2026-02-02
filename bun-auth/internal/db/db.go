package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kartikbazzad/bunbase/pkg/errors"
)

// DB represents the database connection
type DB struct {
	pool *pgxpool.Pool
}

// User model
type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
}

// New creates a new database connection pool and runs migrations
func New(dsn string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	db := &DB{pool: pool}
	if err := db.Migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

// Close closes the database pool
func (db *DB) Close() {
	db.pool.Close()
}

// Migrate creates the necessary tables
func (db *DB) Migrate() error {
	ctx := context.Background()

	// Create Users Table
	_, err := db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			name VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create Sessions Table
	_, err = db.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY,
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			refresh_token VARCHAR(512) NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	return nil
}

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error) {
	id := uuid.New()
	_, err := db.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, name)
		VALUES ($1, $2, $3, $4)
	`, id, email, passwordHash, name)

	if err != nil {
		// Check for unique violation (code 23505)
		// Simplification: just return generic error for now
		return nil, err
	}

	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		CreatedAt:    time.Now(),
	}, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, name, created_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.CreatedAt)

	if err != nil {
		return nil, errors.NotFound("user not found")
	}

	return &user, nil
}
