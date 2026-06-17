package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/VilasGupta/transaction-service/internal/model"
	"github.com/VilasGupta/transaction-service/internal/store"
)

type Handler struct {
	store store.Store
}

func New(s store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req model.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DocumentNumber == "" {
		respondError(w, http.StatusBadRequest, "document_number is required")
		return
	}

	account, err := h.store.CreateAccount(r.Context(), req.DocumentNumber)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	respondJSON(w, http.StatusCreated, account)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("accountId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	account, err := h.store.GetAccount(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			respondError(w, http.StatusNotFound, "account not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respondJSON(w, http.StatusOK, account)
}
