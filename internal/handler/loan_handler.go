package handler

import (
	"errors"
	"net/http"
	"strconv"

	"bank-loan-mvp/internal/audit"
	mw "bank-loan-mvp/internal/middleware"
	"bank-loan-mvp/internal/model"
	"bank-loan-mvp/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type LoanHandler struct {
	loan  *service.LoanService
	audit *audit.Logger
}

func NewLoanHandler(loan *service.LoanService, audit *audit.Logger) *LoanHandler {
	return &LoanHandler{loan: loan, audit: audit}
}

func pageParams(r *http.Request) (int, int) {
	page := 1
	perPage := 20
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if p := r.URL.Query().Get("per_page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 && v <= 100 {
			perPage = v
		}
	}
	return page, perPage
}

func (h *LoanHandler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req model.CreateLoanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	loan, err := h.loan.Create(r.Context(), user.ID, req)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid input")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "loan.created",
		Resource:   "loan_application",
		ResourceID: loan.ID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
	})
	writeSuccess(w, http.StatusCreated, loan)
}

func (h *LoanHandler) ListOwnLoans(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	page, perPage := pageParams(r)
	items, total, err := h.loan.ListOwn(r.Context(), user.ID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	writeSuccessWithMeta(w, http.StatusOK, items, page, perPage, total)
}

func (h *LoanHandler) GetOwnLoan(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	loan, err := h.loan.GetOwn(r.Context(), loanID, user.ID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	writeSuccess(w, http.StatusOK, loan)
}

func (h *LoanHandler) CancelLoan(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	if err := h.loan.Cancel(r.Context(), loanID, user.ID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "loan not found or not cancellable")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "loan.cancelled",
		Resource:   "loan_application",
		ResourceID: loanID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
	})
	writeSuccess(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *LoanHandler) ListAllLoans(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	var filter *string
	if status != "" {
		if status != "pending" && status != "under_review" && status != "approved" && status != "rejected" {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid status filter")
			return
		}
		filter = &status
	}
	page, perPage := pageParams(r)
	items, total, err := h.loan.ListAll(r.Context(), filter, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	writeSuccessWithMeta(w, http.StatusOK, items, page, perPage, total)
}

func (h *LoanHandler) GetAnyLoan(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	loan, err := h.loan.GetAny(r.Context(), loanID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "loan.viewed",
		Resource:   "loan_application",
		ResourceID: loan.ID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
	})
	writeSuccess(w, http.StatusOK, loan)
}

func (h *LoanHandler) DecideLoan(w http.ResponseWriter, r *http.Request) {
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	loanID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req model.DecideLoanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.loan.Decide(r.Context(), loanID, user.ID, req); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid input")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "loan.decision_made",
		Resource:   "loan_application",
		ResourceID: loanID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
		Details:    map[string]any{"decision": req.Decision},
	})
	writeSuccess(w, http.StatusOK, map[string]string{"status": "updated"})
}
