package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Health(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "service unavailable")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok", "db": "ok"})
	}
}
