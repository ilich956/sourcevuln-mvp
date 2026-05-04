package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	FullName     string    `json:"full_name"`
	CreatedAt    time.Time `json:"created_at"`
	IsActive     bool      `json:"is_active"`
}

type LoanApplication struct {
	ID          uuid.UUID `json:"id"`
	ApplicantID uuid.UUID `json:"applicant_id"`
	Amount      float64   `json:"amount"`
	TermMonths  int       `json:"term_months"`
	Purpose     string    `json:"purpose"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type LoanDecision struct {
	ID            uuid.UUID `json:"id"`
	ApplicationID uuid.UUID `json:"application_id"`
	ManagerID     uuid.UUID `json:"manager_id"`
	Decision      string    `json:"decision"`
	Comment       string    `json:"comment,omitempty"`
	DecidedAt     time.Time `json:"decided_at"`
}

type AuditEntry struct {
	ActorID    *uuid.UUID `json:"actor_id,omitempty"`
	Action     string     `json:"action"`
	Resource   string     `json:"resource"`
	ResourceID string     `json:"resource_id,omitempty"`
	IPAddress  string     `json:"ip_address,omitempty"`
	UserAgent  string     `json:"user_agent,omitempty"`
	Details    any        `json:"details,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type AccessClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
}
