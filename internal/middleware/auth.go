package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type ActiveChecker interface {
	IsUserActive(ctx context.Context, id uuid.UUID) (bool, error)
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func Auth(jwtSecret string, checker ActiveChecker) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			tokenStr := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}
				return secret, nil
			})
			if err != nil || !token.Valid {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
				return
			}
			sub, _ := claims["sub"].(string)
			role, _ := claims["role"].(string)
			uid, err := uuid.Parse(sub)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
				return
			}
			active, err := checker.IsUserActive(r.Context(), uid)
			if err != nil || !active {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "account inactive")
				return
			}
			ctx := context.WithValue(r.Context(), userKey, AuthUser{ID: uid, Role: role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
