package model

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

const (
	OperationNormalPurchase      = 1
	OperationPurchaseInstallment = 2
	OperationWithdrawal          = 3
	OperationCreditVoucher       = 4
)

type Account struct {
	AccountID      int64  `json:"account_id"`
	DocumentNumber string `json:"document_number"`
}

type Transaction struct {
	TransactionID   int64           `json:"transaction_id"`
	AccountID       int64           `json:"account_id"`
	OperationTypeID int             `json:"operation_type_id"`
	Amount          decimal.Decimal `json:"amount"`
	Balance         decimal.Decimal `json:"balance"`
	EventDate       time.Time       `json:"event_date"`
}

type CreateAccountRequest struct {
	DocumentNumber string `json:"document_number"`
}

type CreateTransactionRequest struct {
	AccountID       int64           `json:"account_id"`
	OperationTypeID int             `json:"operation_type_id"`
	Amount          decimal.Decimal `json:"amount"`
}

// operationTypeSigns maps each operation type to its sign multiplier: purchases and withdrawals are negative, credit vouchers are positive.
var operationTypeSigns = map[int]decimal.Decimal{
	OperationNormalPurchase:      decimal.NewFromInt(-1),
	OperationPurchaseInstallment: decimal.NewFromInt(-1),
	OperationWithdrawal:          decimal.NewFromInt(-1),
	OperationCreditVoucher:       decimal.NewFromInt(1),
}

// ApplySign forces the correct sign on amount based on operation type. Always uses the absolute value of amount to prevent double-negation.
func ApplySign(operationTypeID int, amount decimal.Decimal) (decimal.Decimal, error) {
	sign, ok := operationTypeSigns[operationTypeID]
	if !ok {
		return decimal.Zero, fmt.Errorf("invalid operation_type_id: %d", operationTypeID)
	}
	return sign.Mul(amount.Abs()), nil
}
