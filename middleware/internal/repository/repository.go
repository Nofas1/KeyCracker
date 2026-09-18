package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"key_cracker/middleware/internal/db"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

type Repo struct {
	pool *pgxpool.Pool
}

var (
	UserNotExist = errors.New("user not found")
)

func NewRepository(logger *slog.Logger) (*Repo, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load .env: %w", err)
	}

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
    conn, err := sqlx.ConnectContext(context.Background(), "pgx", connString)
    if err != nil {
        return nil, fmt.Errorf("repo: fail to sqlx connect: %w", err)
    }
    defer conn.Close()

	migrator := db.NewMigrator(logger, conn)
	if err := migrator.Migrate(); err != nil {
		return nil, fmt.Errorf("repo: failed to run migrations: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("repo: failed to connect to db: %w", err)
	}

	return &Repo{pool: pool}, nil
}

func (rep *Repo) RegisterUser(ctx context.Context, name, hashedPassword string) error {
	query := `INSERT INTO users (name, password) VALUES ($1, $2)`
    _, err := rep.pool.Exec(ctx, query, name, hashedPassword)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("user already exists")
		}
		return fmt.Errorf("failed to register user: %w", err)
	}
	
	return nil
}

func (rep *Repo) GetPasswordHash(ctx context.Context, name string) (string, error) {
	query := `SELECT password FROM users WHERE name = $1`
    var hashed string
    err := rep.pool.QueryRow(ctx, query, name).Scan(&hashed)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return "", fmt.Errorf("user not found")
        }
        return "", fmt.Errorf("failed to get password hash: %w", err)
    }
    return hashed, nil
}

func (rep *Repo) GetToken(ctx context.Context, name string) (string, error) {
	query := `SELECT token FROM users WHERE name = $1`
	var token string
    err := rep.pool.QueryRow(ctx, query, name).Scan(&token)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return "", UserNotExist
        }
        return "", fmt.Errorf("failed to get token: %w", err)
    }
    return token, nil
}
