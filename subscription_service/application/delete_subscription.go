package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"key_cracker/subscription_service/internal/ports"
)

type DeleteSubscriptionUC struct {
	subsRepo ports.SubscriptionRepository
}

func NewDeleteSubscription(subsRepo ports.SubscriptionRepository) *DeleteSubscriptionUC {
	return &DeleteSubscriptionUC{
		subsRepo: subsRepo,
	}
}

func (uc *DeleteSubscriptionUC) Execute(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("app: failed to parse uuid: %w", err)
	}

	err = uc.subsRepo.CancelSubscription(ctx, id)
	if err != nil {
		return fmt.Errorf("app: failed to cancel subscription: %w", err)
	}

	return nil
}
