package middleware

import "net/http"

func RequireRoles(allowed ...string) func(http.Handler) http.Handler {
	allow := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allow[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			if _, ok := allow[u.Role]; !ok {
				writeAuthError(w, http.StatusForbidden, "FORBIDDEN", "access denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
