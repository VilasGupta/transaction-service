package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/VilasGupta/transaction-service/internal/model"
	"github.com/VilasGupta/transaction-service/internal/store"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	store store.Store
}

// New creates a Handler with the given store.
func New(s store.Store) *Handler {
	return &Handler{store: s}
}

// CreateAccount creates a new account.
// @Summary Create account
// @Description Create a new cardholder account with a document number. Returns the existing account with 200 if the document number already exists (idempotent).
// @Tags accounts
// @Accept json
// @Produce json
// @Param request body model.CreateAccountRequest true "Account creation payload"
// @Success 200 {object} model.Account "Account already exists"
// @Success 201 {object} model.Account "Account created"
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /accounts [post]
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	// Decode and validate request body
	var req model.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.DocumentNumber == "" {
		respondError(w, http.StatusBadRequest, "document_number is required")
		return
	}

	// Create account; if document_number already exists, return the existing one (idempotent)
	account, err := h.store.CreateAccount(r.Context(), req.DocumentNumber)
	if err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			respondJSON(w, http.StatusOK, account)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	respondJSON(w, http.StatusCreated, account)
}

// GetAccount retrieves an account by ID.
// @Summary Get account
// @Description Retrieve account details by account ID
// @Tags accounts
// @Produce json
// @Param accountId path int true "Account ID"
// @Success 200 {object} model.Account
// @Failure 400 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /accounts/{accountId} [get]
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	// Parse account ID from URL path
	idStr := r.PathValue("accountId")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	// Fetch account from store
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
