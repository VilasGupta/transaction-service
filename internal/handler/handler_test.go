package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VilasGupta/transaction-service/internal/model"
	"github.com/VilasGupta/transaction-service/internal/store"
)

type mockStore struct {
	createAccountFn     func(ctx context.Context, docNum string) (model.Account, error)
	getAccountFn        func(ctx context.Context, id int64) (model.Account, error)
	createTransactionFn func(ctx context.Context, accountID int64, opTypeID int, amount decimal.Decimal) (model.Transaction, error)
}

func (m *mockStore) CreateAccount(ctx context.Context, docNum string) (model.Account, error) {
	return m.createAccountFn(ctx, docNum)
}

func (m *mockStore) GetAccount(ctx context.Context, id int64) (model.Account, error) {
	return m.getAccountFn(ctx, id)
}

func (m *mockStore) CreateTransaction(ctx context.Context, accountID int64, opTypeID int, amount decimal.Decimal) (model.Transaction, error) {
	return m.createTransactionFn(ctx, accountID, opTypeID, amount)
}

func setupRouter(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /accounts", h.CreateAccount)
	mux.HandleFunc("GET /accounts/{accountId}", h.GetAccount)
	mux.HandleFunc("POST /transactions", h.CreateTransaction)
	return mux
}

func TestCreateAccount(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		mockFn       func(ctx context.Context, docNum string) (model.Account, error)
		wantStatus   int
		wantAccount  *model.Account
		wantErrMsg   string
	}{
		{
			name: "success",
			body: `{"document_number": "12345678900"}`,
			mockFn: func(_ context.Context, docNum string) (model.Account, error) {
				return model.Account{AccountID: 1, DocumentNumber: docNum}, nil
			},
			wantStatus:  http.StatusCreated,
			wantAccount: &model.Account{AccountID: 1, DocumentNumber: "12345678900"},
		},
		{
			name:       "missing document_number",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "document_number is required",
		},
		{
			name:       "invalid json",
			body:       `{invalid}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid request body",
		},
		{
			name: "store error",
			body: `{"document_number": "12345678900"}`,
			mockFn: func(_ context.Context, _ string) (model.Account, error) {
				return model.Account{}, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
			wantErrMsg: "failed to create account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{
				createAccountFn: tt.mockFn,
			}
			if ms.createAccountFn == nil {
				ms.createAccountFn = func(_ context.Context, _ string) (model.Account, error) {
					return model.Account{}, nil
				}
			}

			h := New(ms)
			mux := setupRouter(h)

			req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantAccount != nil {
				var got model.Account
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, *tt.wantAccount, got)
			}

			if tt.wantErrMsg != "" {
				var got errorResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, tt.wantErrMsg, got.Message)
				assert.Equal(t, tt.wantStatus, got.Code)
			}
		})
	}
}

func TestGetAccount(t *testing.T) {
	tests := []struct {
		name        string
		accountID   string
		mockFn      func(ctx context.Context, id int64) (model.Account, error)
		wantStatus  int
		wantAccount *model.Account
		wantErrMsg  string
	}{
		{
			name:      "success",
			accountID: "1",
			mockFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{AccountID: 1, DocumentNumber: "12345678900"}, nil
			},
			wantStatus:  http.StatusOK,
			wantAccount: &model.Account{AccountID: 1, DocumentNumber: "12345678900"},
		},
		{
			name:      "not found",
			accountID: "999",
			mockFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{}, store.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
			wantErrMsg: "account not found",
		},
		{
			name:       "invalid id",
			accountID:  "abc",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid account id",
		},
		{
			name:      "store error",
			accountID: "1",
			mockFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{}, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
			wantErrMsg: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{
				getAccountFn: tt.mockFn,
			}
			if ms.getAccountFn == nil {
				ms.getAccountFn = func(_ context.Context, _ int64) (model.Account, error) {
					return model.Account{}, nil
				}
			}

			h := New(ms)
			mux := setupRouter(h)

			req := httptest.NewRequest(http.MethodGet, "/accounts/"+tt.accountID, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantAccount != nil {
				var got model.Account
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, *tt.wantAccount, got)
			}

			if tt.wantErrMsg != "" {
				var got errorResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, tt.wantErrMsg, got.Message)
			}
		})
	}
}

func TestCreateTransaction(t *testing.T) {
	fixedTime := time.Date(2020, 1, 1, 10, 32, 7, 0, time.UTC)

	tests := []struct {
		name        string
		body        string
		getAccFn    func(ctx context.Context, id int64) (model.Account, error)
		createTxFn  func(ctx context.Context, accountID int64, opTypeID int, amount decimal.Decimal) (model.Transaction, error)
		wantStatus  int
		wantAmount  *decimal.Decimal
		wantErrMsg  string
	}{
		{
			name: "credit voucher stores positive amount",
			body: `{"account_id": 1, "operation_type_id": 4, "amount": 123.45}`,
			getAccFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{AccountID: 1}, nil
			},
			createTxFn: func(_ context.Context, _ int64, _ int, amount decimal.Decimal) (model.Transaction, error) {
				return model.Transaction{
					TransactionID: 1, AccountID: 1, OperationTypeID: 4,
					Amount: amount, EventDate: fixedTime,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantAmount: decPtr(decimal.NewFromFloat(123.45)),
		},
		{
			name: "normal purchase stores negative amount",
			body: `{"account_id": 1, "operation_type_id": 1, "amount": 50.0}`,
			getAccFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{AccountID: 1}, nil
			},
			createTxFn: func(_ context.Context, _ int64, _ int, amount decimal.Decimal) (model.Transaction, error) {
				return model.Transaction{
					TransactionID: 1, AccountID: 1, OperationTypeID: 1,
					Amount: amount, EventDate: fixedTime,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantAmount: decPtr(decimal.NewFromFloat(-50.0)),
		},
		{
			name: "purchase with installments stores negative amount",
			body: `{"account_id": 1, "operation_type_id": 2, "amount": 23.5}`,
			getAccFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{AccountID: 1}, nil
			},
			createTxFn: func(_ context.Context, _ int64, _ int, amount decimal.Decimal) (model.Transaction, error) {
				return model.Transaction{
					TransactionID: 1, AccountID: 1, OperationTypeID: 2,
					Amount: amount, EventDate: fixedTime,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantAmount: decPtr(decimal.NewFromFloat(-23.5)),
		},
		{
			name: "withdrawal stores negative amount",
			body: `{"account_id": 1, "operation_type_id": 3, "amount": 18.7}`,
			getAccFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{AccountID: 1}, nil
			},
			createTxFn: func(_ context.Context, _ int64, _ int, amount decimal.Decimal) (model.Transaction, error) {
				return model.Transaction{
					TransactionID: 1, AccountID: 1, OperationTypeID: 3,
					Amount: amount, EventDate: fixedTime,
				}, nil
			},
			wantStatus: http.StatusCreated,
			wantAmount: decPtr(decimal.NewFromFloat(-18.7)),
		},
		{
			name:       "invalid json",
			body:       `{bad}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid request body",
		},
		{
			name:       "missing account_id",
			body:       `{"operation_type_id": 1, "amount": 10}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "account_id is required",
		},
		{
			name:       "zero amount",
			body:       `{"account_id": 1, "operation_type_id": 1, "amount": 0}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "amount must be greater than zero",
		},
		{
			name:       "negative amount",
			body:       `{"account_id": 1, "operation_type_id": 1, "amount": -10}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "amount must be greater than zero",
		},
		{
			name:       "invalid operation type",
			body:       `{"account_id": 1, "operation_type_id": 99, "amount": 10}`,
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "invalid operation_type_id",
		},
		{
			name: "account not found",
			body: `{"account_id": 999, "operation_type_id": 1, "amount": 10}`,
			getAccFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{}, store.ErrNotFound
			},
			wantStatus: http.StatusNotFound,
			wantErrMsg: "account not found",
		},
		{
			name: "store create transaction error",
			body: `{"account_id": 1, "operation_type_id": 1, "amount": 10}`,
			getAccFn: func(_ context.Context, _ int64) (model.Account, error) {
				return model.Account{AccountID: 1}, nil
			},
			createTxFn: func(_ context.Context, _ int64, _ int, _ decimal.Decimal) (model.Transaction, error) {
				return model.Transaction{}, assert.AnError
			},
			wantStatus: http.StatusInternalServerError,
			wantErrMsg: "failed to create transaction",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &mockStore{
				getAccountFn: tt.getAccFn,
				createTransactionFn: tt.createTxFn,
			}
			if ms.getAccountFn == nil {
				ms.getAccountFn = func(_ context.Context, _ int64) (model.Account, error) {
					return model.Account{}, nil
				}
			}
			if ms.createTransactionFn == nil {
				ms.createTransactionFn = func(_ context.Context, _ int64, _ int, _ decimal.Decimal) (model.Transaction, error) {
					return model.Transaction{}, nil
				}
			}

			h := New(ms)
			mux := setupRouter(h)

			req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)

			if tt.wantAmount != nil {
				var got model.Transaction
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.True(t, tt.wantAmount.Equal(got.Amount), "expected amount %s, got %s", tt.wantAmount, got.Amount)
			}

			if tt.wantErrMsg != "" {
				var got errorResponse
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
				assert.Equal(t, tt.wantErrMsg, got.Message)
				assert.Equal(t, tt.wantStatus, got.Code)
			}
		})
	}
}

func decPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}
