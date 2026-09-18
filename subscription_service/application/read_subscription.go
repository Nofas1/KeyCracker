package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"key_cracker/subscription_service/internal/ports"
	"key_cracker/subscription_service/internal/domain"
)

type ReadSubscriptionUC struct {
	subsRepo ports.SubscriptionRepository
}

func NewReadSubscription(subsRepo ports.SubscriptionRepository) *ReadSubscriptionUC {
	return &ReadSubscriptionUC{
		subsRepo: subsRepo,
	}
}

func (uc *ReadSubscriptionUC) Execute(ctx context.Context, userID string) (*domain.Subscription, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("app: failed to parse uuid: %w", err)
	}

	subscriptionInfo, err := uc.subsRepo.GetActive(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("app: failed to check subscription status: %w", err)
	}

	return subscriptionInfo, nil
}
