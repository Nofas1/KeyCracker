package ports

import (
	"context"

	"github.com/google/uuid"
	"key_cracker/subscription_service/internal/domain"
)

type SubscriptionRepository interface {
	CreateSubscription(ctx context.Context, userInfo *domain.Subscription) error
	GetActive(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	CancelSubscription(ctx context.Context, userID uuid.UUID) error
	GetPlan(ctx context.Context, planName string) (*domain.Plan, error)
}
