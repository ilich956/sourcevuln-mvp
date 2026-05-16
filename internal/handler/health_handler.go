package handler

import (
	"database/sql"
	"net/http"
)

func Health(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.PingContext(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "service unavailable")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
	}
}
