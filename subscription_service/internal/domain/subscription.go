package domain

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID        int64
	UserID    uuid.UUID
	PlanID    int64
	StartAt   time.Time
	ExpiredAt time.Time
	Status    string
}
