package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/VilasGupta/transaction-service/internal/model"
	"github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
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
	//fetch all relevant transactions for credit
	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Transaction{}, err
	}

	defer dbTx.Rollback()
	paymentBalance := amount

	if operationTypeID == model.OperationCreditVoucher {
		dischargeTransactions, err := dbTx.QueryContext(ctx,
			"SELECT transaction_id, balance from transactions where account_id = ? and operation_type_id != ? and balance < 0 for update", accountID, model.OperationCreditVoucher)

		if err != nil {
			return model.Transaction{}, err
		}

		var pending []model.Transaction
		for dischargeTransactions.Next() {
			var dischargeTransaction model.Transaction
			err := dischargeTransactions.Scan(&dischargeTransaction.TransactionID, &dischargeTransaction.Balance)
			if err != nil {
				dischargeTransactions.Close()
				return model.Transaction{}, err
			}
			pending = append(pending, dischargeTransaction)
		}
		if err := dischargeTransactions.Err(); err != nil {
			dischargeTransactions.Close()
			return model.Transaction{}, err
		}
		dischargeTransactions.Close()

		for _, dischargeTransaction := range pending {
			var newBalance decimal.Decimal
			var dischargeBalance decimal.Decimal

			//fully able to discharge the transaction
			if amount.IsPositive() {
				if amount.GreaterThan(dischargeTransaction.Balance.Abs()) { //60 > 50
					newBalance = dischargeTransaction.Balance.Add(amount) // -10
					paymentBalance = newBalance.Abs()                     //10
					dischargeBalance = decimal.Zero                       //0
				} else {
					newBalance = dischargeTransaction.Balance.Add(paymentBalance) //-23.5 + 10 = 13.5
					paymentBalance = decimal.Zero                                 //0
					dischargeBalance = newBalance                                 //-13
				}
			}

			_, err = dbTx.ExecContext(ctx, "UPDATE transactions set balance = ? where transaction_id = ?", dischargeBalance, dischargeTransaction.TransactionID)
			if err != nil {
				return model.Transaction{}, err
			}
		}
	}

	// Insert with StringFixed(2) to match the DECIMAL(15,2) column precision
	result, err := dbTx.ExecContext(ctx,
		"INSERT INTO transactions (account_id, operation_type_id, amount, balance) VALUES (?, ?, ?, ?)",
		accountID, operationTypeID, amount.StringFixed(2), paymentBalance.StringFixed(2))
	if err != nil {
		return model.Transaction{}, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return model.Transaction{}, err
	}

	err = dbTx.Commit()
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
