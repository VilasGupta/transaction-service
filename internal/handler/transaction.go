package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/shopspring/decimal"
	"github.com/VilasGupta/transaction-service/internal/model"
	"github.com/VilasGupta/transaction-service/internal/store"
)

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AccountID <= 0 {
		respondError(w, http.StatusBadRequest, "account_id is required")
		return
	}

	if req.Amount.LessThanOrEqual(decimal.Zero) {
		respondError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}

	signedAmount, err := model.ApplySign(req.OperationTypeID, req.Amount)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid operation_type_id")
		return
	}

	_, err = h.store.GetAccount(r.Context(), req.AccountID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "account not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	tx, err := h.store.CreateTransaction(r.Context(), req.AccountID, req.OperationTypeID, signedAmount)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create transaction")
		return
	}

	respondJSON(w, http.StatusCreated, tx)
}
