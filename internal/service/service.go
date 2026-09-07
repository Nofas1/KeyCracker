package service

import (
	"context"
	"fmt"
	"key_cracker/internal"
	"key_cracker/internal/mongostore"
)

type KeyCrackerService struct{
	locks *mongostore.LockStore
}

func NewKeyCrackerService(locks *mongostore.LockStore) *KeyCrackerService {
	return &KeyCrackerService{locks: locks}
}

func (s *KeyCrackerService) Solve(initialState string, relations [][]int) ([]internal.ParentData, error) {
	key_cracker := internal.Constructor(relations, initialState)
	key_cracker.BuildTree()
	ans := key_cracker.Answer()
	return ans, nil
}

func (s *KeyCrackerService) CreateLock(ctx context.Context, name, initialState string, relations [][]int) (*mongostore.Lock, error) {
	return s.locks.Create(ctx, name, initialState, relations)
}

func (s *KeyCrackerService) GetLock(ctx context.Context, id string) (*mongostore.Lock, error) {
	return s.locks.Get(ctx, id)
}

func (s *KeyCrackerService) ListLocks(ctx context.Context) ([]mongostore.Lock, error) {
	return s.locks.List(ctx)
}

func (s *KeyCrackerService) SolveLock(ctx context.Context, id string) ([]internal.ParentData, error) {
	lock, err := s.locks.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("service: load lock: %w", err)
	}
	res, err := s.Solve(lock.InitialState, lock.Relations)
	if err != nil {
		return []internal.ParentData{}, fmt.Errorf("service: failed to solve: %w", err)
	}
	return res, nil
}

func (s *KeyCrackerService) PingMongo(ctx context.Context) error {
	return s.locks.Ping(ctx)
}
