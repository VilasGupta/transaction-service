package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/shopspring/decimal"
	"github.com/VilasGupta/transaction-service/internal/model"
	"github.com/VilasGupta/transaction-service/internal/store"
)

// CreateTransaction creates a new transaction.
// @Summary Create transaction
// @Description Create a financial transaction for an account. Amount must be positive; the service applies the correct sign based on operation type (1: Normal Purchase, 2: Purchase with installments, 3: Withdrawal, 4: Credit Voucher).
// @Tags transactions
// @Accept json
// @Produce json
// @Param request body model.CreateTransactionRequest true "Transaction creation payload"
// @Success 201 {object} model.Transaction
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /transactions [post]
func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request body
	var req model.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AccountID <= 0 {
		respondError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	// Client always sends a positive amount; reject zero or negative
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		respondError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}

	// Apply the correct sign based on operation type (negative for purchases/withdrawals, positive for credit vouchers)
	signedAmount, err := model.ApplySign(req.OperationTypeID, req.Amount)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid operation_type_id")
		return
	}

	// Verify the account exists before creating the transaction
	_, err = h.store.GetAccount(r.Context(), req.AccountID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "account not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Persist transaction with the signed amount
	tx, err := h.store.CreateTransaction(r.Context(), req.AccountID, req.OperationTypeID, signedAmount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}

	respondJSON(w, http.StatusCreated, tx)
}
