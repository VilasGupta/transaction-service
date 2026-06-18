package store

import (
	"context"
	"errors"

	"github.com/shopspring/decimal"
	"github.com/VilasGupta/transaction-service/internal/model"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate")
)

// Store defines the persistence operations for accounts and transactions.
type Store interface {
	CreateAccount(ctx context.Context, documentNumber string) (model.Account, error)
	GetAccount(ctx context.Context, id int64) (model.Account, error)
	CreateTransaction(ctx context.Context, accountID int64, operationTypeID int, amount decimal.Decimal) (model.Transaction, error)
}
