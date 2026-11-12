package banking

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var ctx = context.Background()

func TestCreateAccount(t *testing.T) {
	svc := NewService()

	tests := []struct {
		name           string
		initialDeposit int64
		wantErr        error
	}{
		{"zero deposit", 0, nil},
		{"positive deposit", 500, nil},
		{"negative deposit", -1, ErrInvalidAmount},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, err := svc.CreateAccount(ctx, tc.initialDeposit)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got err %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil && id == "" {
				t.Fatal("expected non empty account ID")
			}
			if tc.wantErr == nil {
				bal, _ := svc.Balance(ctx, id)
				if bal != tc.initialDeposit {
					t.Fatalf("got balance %d, want %d", bal, tc.initialDeposit)
				}
			}
		})
	}
}

func TestCreateAccount_CancelledContext(t *testing.T) {
	svc := NewService()
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.CreateAccount(cancelled, 100)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestDeposit(t *testing.T) {
	svc := NewService()
	id, _ := svc.CreateAccount(ctx, 100)

	tests := []struct {
		name    string
		id      string
		amount  int64
		wantBal int64
		wantErr error
	}{
		{"valid deposit", id, 200, 300, nil},
		{"zero amount", id, 0, 0, ErrInvalidAmount},
		{"negative amount", id, -50, 0, ErrInvalidAmount},
		{"unknown account", "bad id", 100, 0, ErrAccountNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.Deposit(ctx, tc.id, tc.amount)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got err %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				bal, _ := svc.Balance(ctx, tc.id)
				if bal != tc.wantBal {
					t.Fatalf("got balance %d, want %d", bal, tc.wantBal)
				}
			}
		})
	}
}

func TestWithdraw(t *testing.T) {
	svc := NewService()
	id, _ := svc.CreateAccount(ctx, 500)

	tests := []struct {
		name    string
		id      string
		amount  int64
		wantBal int64
		wantErr error
	}{
		{"valid withdrawal", id, 200, 300, nil},
		{"exact balance", id, 300, 0, nil},
		{"insufficient funds", id, 1, 0, ErrInsufficientFunds},
		{"zero amount", id, 0, 0, ErrInvalidAmount},
		{"negative amount", id, -10, 0, ErrInvalidAmount},
		{"unknown account", "bad id", 100, 0, ErrAccountNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.Withdraw(ctx, tc.id, tc.amount)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got err %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				bal, _ := svc.Balance(ctx, tc.id)
				if bal != tc.wantBal {
					t.Fatalf("got balance %d, want %d", bal, tc.wantBal)
				}
			}
		})
	}
}

func TestTransfer(t *testing.T) {
	svc := NewService()
	a, _ := svc.CreateAccount(ctx, 1000)
	b, _ := svc.CreateAccount(ctx, 0)

	tests := []struct {
		name     string
		from, to string
		amount   int64
		wantA    int64
		wantB    int64
		wantErr  error
	}{
		{"valid transfer", a, b, 400, 600, 400, nil},
		{"insufficient funds", a, b, 10000, 0, 0, ErrInsufficientFunds},
		{"same account", a, a, 100, 0, 0, ErrSameAccount},
		{"zero amount", a, b, 0, 0, 0, ErrInvalidAmount},
		{"unknown from", "bad", b, 100, 0, 0, ErrAccountNotFound},
		{"unknown to", a, "bad", 100, 0, 0, ErrAccountNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.Transfer(ctx, tc.from, tc.to, tc.amount)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got err %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				balA, _ := svc.Balance(ctx, tc.from)
				balB, _ := svc.Balance(ctx, tc.to)
				if balA != tc.wantA || balB != tc.wantB {
					t.Fatalf("got balances (%d, %d), want (%d, %d)", balA, balB, tc.wantA, tc.wantB)
				}
			}
		})
	}
}

func TestBalance(t *testing.T) {
	svc := NewService()
	id, _ := svc.CreateAccount(ctx, 750)

	bal, err := svc.Balance(ctx, id)
	if err != nil || bal != 750 {
		t.Fatalf("got (%d, %v), want (750, nil)", bal, err)
	}

	_, err = svc.Balance(ctx, "nonexistent")
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("got %v, want ErrAccountNotFound", err)
	}
}

func TestConcurrentSafety(t *testing.T) {
	svc := NewService()
	id, _ := svc.CreateAccount(ctx, 10000)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = svc.Deposit(ctx, id, 10)
		}()
		go func() {
			defer wg.Done()
			_ = svc.Withdraw(ctx, id, 10)
		}()
	}
	wg.Wait()

	bal, err := svc.Balance(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if bal != 10000 {
		t.Fatalf("got balance %d after equal deposits/withdrawals, want 10000", bal)
	}
}
