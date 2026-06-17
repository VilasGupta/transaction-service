package store

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"
	"github.com/VilasGupta/transaction-service/internal/model"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) CreateAccount(ctx context.Context, documentNumber string) (model.Account, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO accounts (document_number) VALUES (?)", documentNumber)
	if err != nil {
		return model.Account{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.Account{}, err
	}

	return model.Account{
		AccountID:      id,
		DocumentNumber: documentNumber,
	}, nil
}

func (s *MySQLStore) GetAccount(ctx context.Context, id int64) (model.Account, error) {
	var account model.Account
	err := s.db.QueryRowContext(ctx,
		"SELECT account_id, document_number FROM accounts WHERE account_id = ?", id,
	).Scan(&account.AccountID, &account.DocumentNumber)

	if err == sql.ErrNoRows {
		return model.Account{}, ErrNotFound
	}
	return account, err
}

func (s *MySQLStore) CreateTransaction(ctx context.Context, accountID int64, operationTypeID int, amount decimal.Decimal) (model.Transaction, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO transactions (account_id, operation_type_id, amount) VALUES (?, ?, ?)",
		accountID, operationTypeID, amount.StringFixed(2))
	if err != nil {
		return model.Transaction{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.Transaction{}, err
	}

	var tx model.Transaction
	err = s.db.QueryRowContext(ctx,
		"SELECT transaction_id, account_id, operation_type_id, amount, event_date FROM transactions WHERE transaction_id = ?", id,
	).Scan(&tx.TransactionID, &tx.AccountID, &tx.OperationTypeID, &tx.Amount, &tx.EventDate)

	return tx, err
}
