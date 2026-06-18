package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"github.com/VilasGupta/transaction-service/internal/model"
)

// MySQLStore implements Store using a MySQL database.
type MySQLStore struct {
	db *sql.DB
}

// NewMySQLStore returns a new MySQLStore backed by the given connection pool.
func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

// CreateAccount inserts a new account. Returns the existing account with ErrDuplicate if the document number is already taken.
func (s *MySQLStore) CreateAccount(ctx context.Context, documentNumber string) (model.Account, error) {
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO accounts (document_number) VALUES (?)", documentNumber)
	if err != nil {
		// Duplicate document_number — fetch and return the existing account for idempotency
		if isDuplicateKey(err) {
			var account model.Account
			qErr := s.db.QueryRowContext(ctx,
				"SELECT account_id, document_number FROM accounts WHERE document_number = ?", documentNumber,
			).Scan(&account.AccountID, &account.DocumentNumber)
			if qErr != nil {
				return model.Account{}, qErr
			}
			return account, ErrDuplicate
		}
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

// isDuplicateKey checks for MySQL error 1062 (duplicate entry).
func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}

// GetAccount retrieves an account by ID. Returns ErrNotFound if it doesn't exist.
func (s *MySQLStore) GetAccount(ctx context.Context, id int64) (model.Account, error) {
	var account model.Account
	err := s.db.QueryRowContext(ctx,
		"SELECT account_id, document_number FROM accounts WHERE account_id = ?", id,
	).Scan(&account.AccountID, &account.DocumentNumber)

	// Map sql.ErrNoRows to our sentinel so the handler can return 404
	if err == sql.ErrNoRows {
		return model.Account{}, ErrNotFound
	}
	return account, err
}

// CreateTransaction inserts a transaction and re-queries it to capture the server-generated event_date.
func (s *MySQLStore) CreateTransaction(ctx context.Context, accountID int64, operationTypeID int, amount decimal.Decimal) (model.Transaction, error) {
	// Insert with StringFixed(2) to match the DECIMAL(15,2) column precision
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

	// Re-query to get the server-generated event_date (DEFAULT CURRENT_TIMESTAMP)
	var tx model.Transaction
	err = s.db.QueryRowContext(ctx,
		"SELECT transaction_id, account_id, operation_type_id, amount, event_date FROM transactions WHERE transaction_id = ?", id,
	).Scan(&tx.TransactionID, &tx.AccountID, &tx.OperationTypeID, &tx.Amount, &tx.EventDate)

	return tx, err
}
