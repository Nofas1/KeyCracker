package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"key_cracker/subscription_service/internal/ports"
)

type CreateSubscriptionUC struct {
	subsRepo ports.SubscriptionRepository
}

func NewCreateSubscription(subsRepo ports.SubscriptionRepository) *CreateSubscriptionUC {
	return &CreateSubscriptionUC{
		subsRepo: subsRepo,
	}
}

func (uc *CreateSubscriptionUC) Execute(ctx context.Context, planName, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("app: failed to parse uuid: %w", err)
	}

	subscriptionInfo, err := uc.subsRepo.GetActive(ctx, id)
	if err != nil {
		return fmt.Errorf("app: failed to check subscription status: %w", err)
	}

	currentPlan, err := uc.subsRepo.GetPlan(ctx, planName)
	if err != nil {
		return fmt.Errorf("app: failed to get the plan of the user: %w", err)
	}

	if subscriptionInfo.Status != "active" {
		subscriptionInfo.StartAt = time.Now()
		subscriptionInfo.Status = "active"
	} else {
		subscriptionInfo.StartAt = subscriptionInfo.ExpiredAt
	}
	
	subscriptionInfo.PlanID = currentPlan.ID
	subscriptionInfo.ExpiredAt = subscriptionInfo.StartAt.AddDate(0, 0, int(currentPlan.Duration))
	
	err = uc.subsRepo.CreateSubscription(ctx, subscriptionInfo)
	if err != nil {
		return fmt.Errorf("app: failed to create subscription: %w", err)
	}
	return nil
}
