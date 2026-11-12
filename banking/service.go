package banking

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type service struct {
	mu       sync.RWMutex
	accounts map[string]*Account
}

func NewService() Service {
	return &service{accounts: make(map[string]*Account)}
}

func (s *service) CreateAccount(ctx context.Context, initialDeposit int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if initialDeposit < 0 {
		return "", ErrInvalidAmount
	}

	id, err := generateID()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.accounts[id] = &Account{ID: id, balance: initialDeposit}
	return id, nil
}

func (s *service) Deposit(ctx context.Context, accountID string, amount int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[accountID]
	if !ok {
		return ErrAccountNotFound
	}

	acc.balance += amount
	return nil
}

func (s *service) Withdraw(ctx context.Context, accountID string, amount int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[accountID]
	if !ok {
		return ErrAccountNotFound
	}
	if acc.balance < amount {
		return ErrInsufficientFunds
	}

	acc.balance -= amount
	return nil
}

func (s *service) Transfer(ctx context.Context, fromID, toID string, amount int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if fromID == toID {
		return ErrSameAccount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	from, ok := s.accounts[fromID]
	if !ok {
		return ErrAccountNotFound
	}
	to, ok := s.accounts[toID]
	if !ok {
		return ErrAccountNotFound
	}
	if from.balance < amount {
		return ErrInsufficientFunds
	}

	from.balance -= amount
	to.balance += amount
	return nil
}

func (s *service) Balance(ctx context.Context, accountID string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, ok := s.accounts[accountID]
	if !ok {
		return 0, ErrAccountNotFound
	}
	return acc.balance, nil
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
