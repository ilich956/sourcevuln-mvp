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

type AdminHandler struct {
	admin *service.AdminService
	audit *audit.Logger
}

func NewAdminHandler(admin *service.AdminService, audit *audit.Logger) *AdminHandler {
	return &AdminHandler{admin: admin, audit: audit}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
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
	users, total, err := h.admin.ListUsers(r.Context(), page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	resp := make([]model.UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, model.UserResponse{
			ID:        u.ID.String(),
			Email:     u.Email,
			Role:      u.Role,
			FullName:  u.FullName,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeSuccessWithMeta(w, http.StatusOK, resp, page, perPage, total)
}

func (h *AdminHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.admin.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	writeSuccess(w, http.StatusOK, stats)
}

func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req model.UpdateUserRoleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.admin.SetUserRole(r.Context(), userID, req.Role); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &actor.ID,
		Action:     "admin.user_role_changed",
		Resource:   "user",
		ResourceID: userID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
		Details:    map[string]any{"role": req.Role},
	})
	writeSuccess(w, http.StatusOK, map[string]any{"id": userID.String(), "role": req.Role})
}

func (h *AdminHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req model.UpdateUserStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.admin.SetUserStatus(r.Context(), userID, req.IsActive); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "unable to process request")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &actor.ID,
		Action:     "admin.user_status_changed",
		Resource:   "user",
		ResourceID: userID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
		Details:    map[string]any{"is_active": req.IsActive},
	})
	writeSuccess(w, http.StatusOK, map[string]any{"id": userID.String(), "is_active": req.IsActive})
}
