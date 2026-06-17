package model

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplySign(t *testing.T) {
	tests := []struct {
		name            string
		operationTypeID int
		amount          decimal.Decimal
		expected        decimal.Decimal
		wantErr         bool
	}{
		{
			name:            "normal purchase stores negative",
			operationTypeID: OperationNormalPurchase,
			amount:          decimal.NewFromFloat(50.0),
			expected:        decimal.NewFromFloat(-50.0),
		},
		{
			name:            "purchase with installments stores negative",
			operationTypeID: OperationPurchaseInstallment,
			amount:          decimal.NewFromFloat(23.5),
			expected:        decimal.NewFromFloat(-23.5),
		},
		{
			name:            "withdrawal stores negative",
			operationTypeID: OperationWithdrawal,
			amount:          decimal.NewFromFloat(18.7),
			expected:        decimal.NewFromFloat(-18.7),
		},
		{
			name:            "credit voucher stores positive",
			operationTypeID: OperationCreditVoucher,
			amount:          decimal.NewFromFloat(60.0),
			expected:        decimal.NewFromFloat(60.0),
		},
		{
			name:            "negative input still gets correct sign for purchase",
			operationTypeID: OperationNormalPurchase,
			amount:          decimal.NewFromFloat(-100.0),
			expected:        decimal.NewFromFloat(-100.0),
		},
		{
			name:            "negative input still gets correct sign for credit voucher",
			operationTypeID: OperationCreditVoucher,
			amount:          decimal.NewFromFloat(-100.0),
			expected:        decimal.NewFromFloat(100.0),
		},
		{
			name:            "invalid operation type returns error",
			operationTypeID: 5,
			amount:          decimal.NewFromFloat(10.0),
			wantErr:         true,
		},
		{
			name:            "zero operation type returns error",
			operationTypeID: 0,
			amount:          decimal.NewFromFloat(10.0),
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ApplySign(tt.operationTypeID, tt.amount)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, tt.expected.Equal(result), "expected %s, got %s", tt.expected, result)
		})
	}
}
