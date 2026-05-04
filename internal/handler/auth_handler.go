package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bank-loan-mvp/internal/audit"
	mw "bank-loan-mvp/internal/middleware"
	"bank-loan-mvp/internal/model"
	"bank-loan-mvp/internal/service"
)

type AuthHandler struct {
	auth  *service.AuthService
	audit *audit.Logger
}

func NewAuthHandler(auth *service.AuthService, audit *audit.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, audit: audit}
}

func maskedEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	return parts[0] + "@***"
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	u, err := h.auth.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			writeError(w, http.StatusBadRequest, "ALREADY_EXISTS", "unable to process request")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid input")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &u.ID,
		Action:     "user.register",
		Resource:   "user",
		ResourceID: u.ID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
		Details:    map[string]any{"email": maskedEmail(u.Email)},
	})
	writeSuccess(w, http.StatusCreated, model.UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Role:      u.Role,
		FullName:  u.FullName,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	tokens, user, err := h.auth.Login(r.Context(), req)
	if err != nil {
		action := "user.login_failed"
		h.audit.Log(r.Context(), model.AuditEntry{
			Action:    action,
			Resource:  "auth",
			IPAddress: r.RemoteAddr,
			UserAgent: r.UserAgent(),
			Details:   map[string]any{"email": maskedEmail(req.Email)},
		})
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid input")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "user.login",
		Resource:   "auth",
		ResourceID: user.ID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
		Details:    map[string]any{"email": maskedEmail(user.Email)},
	})
	writeSuccess(w, http.StatusOK, tokens)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	resp, user, err := h.auth.Refresh(r.Context(), req)
	if err != nil {
		h.audit.Log(r.Context(), model.AuditEntry{
			Action:    "user.token_refresh_failed",
			Resource:  "auth",
			IPAddress: mw.ClientIP(r),
			UserAgent: r.UserAgent(),
		})
		if errors.Is(err, service.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token")
			return
		}
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid input")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "user.token_refreshed",
		Resource:   "auth",
		ResourceID: user.ID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
	})
	writeSuccess(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req model.LogoutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	user, ok := mw.UserFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	if err := h.auth.Logout(r.Context(), user.ID, req); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "invalid input")
		return
	}
	h.audit.Log(r.Context(), model.AuditEntry{
		ActorID:    &user.ID,
		Action:     "user.logout",
		Resource:   "auth",
		ResourceID: user.ID.String(),
		IPAddress:  mw.ClientIP(r),
		UserAgent:  r.UserAgent(),
	})
	writeSuccess(w, http.StatusOK, map[string]string{"message": "logged out"})
}
