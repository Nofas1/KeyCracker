package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"key_cracker/subscription_service/internal/db"
	"key_cracker/subscription_service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

type Repo struct {
	pool *pgxpool.Pool
}

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

func (rep *Repo) CreateSubscription(ctx context.Context, userInfo *domain.Subscription) error {
	query := `INSERT INTO subscriptions (user_id, plan_id, start_at, expired_at, status) VALUES ($1, $2, $3, $4, $5)`
	if _, err := rep.pool.Exec(ctx, query, userInfo.UserID, userInfo.PlanID, userInfo.StartAt, userInfo.ExpiredAt, userInfo.Status); err != nil {
		return fmt.Errorf("repo: failed to initialize subscription: %w", err)
	}
	return nil
}

func (rep *Repo) GetActive(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	query := `
        SELECT plan_id, start_at, expired_at, status
        FROM subscriptions WHERE user_id = $1
	`

	var plan_id int64
	var start_at, expired_at time.Time
	var status string

	err := rep.pool.QueryRow(ctx, query, userID).Scan(
		&plan_id, &start_at, &expired_at, &status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.Subscription{}, nil
		}
		return &domain.Subscription{}, fmt.Errorf("repo: failed to get subscription: %w", err)
	}

	return &domain.Subscription{
		UserID: userID,
		PlanID: plan_id,
		StartAt: start_at,
		ExpiredAt: expired_at,
		Status: status,
	}, nil
}

func (rep *Repo) CancelSubscription(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE subscriptions
		SET status = 'cancelled', expired_at = NOW()
		WHERE user_id = $1 AND status = 'active'
	`

	tag, err := rep.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("repo: failed to cancel subscription: %w", err)
	}
	
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("repo: no active subscription found for user %s", userID)
	}
	
	return nil
}

func (rep *Repo) GetPlan(ctx context.Context, planName string) (*domain.Plan, error) {
	query := `
		SELECT duration
		FROM plans WHERE name = $1
	`
	var duration int64

	err := rep.pool.QueryRow(ctx, query, planName).Scan(
		&duration,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.Plan{}, nil
		}
		return &domain.Plan{}, fmt.Errorf("repo: failed to get plan: %w", err)
	}

	return &domain.Plan{
		Duration: duration,
	}, nil

}
