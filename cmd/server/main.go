package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"bank-loan-mvp/internal/audit"
	"bank-loan-mvp/migrations"

	"github.com/joho/godotenv"

	"bank-loan-mvp/internal/config"
	"bank-loan-mvp/internal/db"
	"bank-loan-mvp/internal/handler"
	imw "bank-loan-mvp/internal/middleware"
	"bank-loan-mvp/internal/repository"
	"bank-loan-mvp/internal/service"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-playground/validator/v10"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	pool, err := db.Connect(context.Background(), cfg.DBPath, migrations.UpFiles)
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	validate := validator.New(validator.WithRequiredStructEnabled())
	auditLogger := audit.New(repo)
	authSvc := service.NewAuthService(repo, validate, cfg.JWTSecret)
	loanSvc := service.NewLoanService(repo, validate)
	adminSvc := service.NewAdminService(repo)

	authHandler := handler.NewAuthHandler(authSvc, auditLogger)
	loanHandler := handler.NewLoanHandler(loanSvc, auditLogger)
	adminHandler := handler.NewAdminHandler(adminSvc, auditLogger)

	r := chi.NewRouter()

	globalLimiter := imw.NewIPRateLimiter(100, 100)
	authLimiter := imw.NewIPRateLimiter(20, 20)
	allowedOrigins := parseAllowedOrigins()

	r.Use(imw.SecurityHeaders)
	r.Use(chimw.RequestID)
	r.Use(chimw.Logger)
	r.Use(globalLimiter.Middleware)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(imw.BodyLimit(1 << 20))

	r.Get("/health", handler.Health(pool))

	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/auth", func(auth chi.Router) {
			auth.Use(authLimiter.Middleware)
			auth.Post("/register", authHandler.Register)
			auth.Post("/login", authHandler.Login)
			auth.Post("/refresh", authHandler.Refresh)

			auth.Group(func(protected chi.Router) {
				protected.Use(imw.Auth(cfg.JWTSecret, repo))
				protected.Post("/logout", authHandler.Logout)
			})
		})

		api.Group(func(client chi.Router) {
			client.Use(imw.Auth(cfg.JWTSecret, repo))
			client.Use(imw.RequireRoles("client"))
			client.Post("/loans", loanHandler.CreateLoan)
			client.Get("/loans", loanHandler.ListOwnLoans)
			client.Get("/loans/{id}", loanHandler.GetOwnLoan)
			client.Patch("/loans/{id}/cancel", loanHandler.CancelLoan)
		})

		api.Group(func(manager chi.Router) {
			manager.Use(imw.Auth(cfg.JWTSecret, repo))
			manager.Use(imw.RequireRoles("manager", "admin"))
			manager.Get("/manager/loans", loanHandler.ListAllLoans)
			manager.Get("/manager/loans/{id}", loanHandler.GetAnyLoan)
			manager.Patch("/manager/loans/{id}/decide", loanHandler.DecideLoan)
			manager.Get("/manager/stats", adminHandler.GetStats)
		})

		api.Group(func(admin chi.Router) {
			admin.Use(imw.Auth(cfg.JWTSecret, repo))
			admin.Use(imw.RequireRoles("admin"))
			admin.Get("/admin/users", adminHandler.ListUsers)
			admin.Patch("/admin/users/{id}/status", adminHandler.UpdateUserStatus)
			admin.Patch("/admin/users/{id}/role", adminHandler.UpdateUserRole)
		})
	})

	log.Printf("Server starting on port %s", cfg.Port)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func parseAllowedOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if raw == "" {
		// Reasonable dev defaults for running the static frontend locally.
		return []string{"http://localhost:3000", "http://localhost", "http://127.0.0.1:3000", "http://127.0.0.1"}
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			origins = append(origins, p)
		}
	}
	if len(origins) == 0 {
		return []string{"http://localhost:3000", "http://localhost", "http://127.0.0.1:3000", "http://127.0.0.1"}
	}
	return origins
}
