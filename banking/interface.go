package banking

import "context"

type Service interface {
	CreateAccount(ctx context.Context, initialDeposit int64) (string, error)
	Deposit(ctx context.Context, accountID string, amount int64) error
	Withdraw(ctx context.Context, accountID string, amount int64) error
	Transfer(ctx context.Context, fromID, toID string, amount int64) error
	Balance(ctx context.Context, accountID string) (int64, error)
}
