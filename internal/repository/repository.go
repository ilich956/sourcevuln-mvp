package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"bank-loan-mvp/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConflict = errors.New("conflict")

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func mapPGErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrConflict
	}
	return err
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, role, fullName string) (model.User, error) {
	q := `INSERT INTO users (email, password_hash, role, full_name)
	      VALUES ($1, $2, $3, $4)
	      RETURNING id::text, email, password_hash, role, full_name, created_at, is_active`
	var idStr string
	var u model.User
	err := r.db.QueryRow(ctx, q, email, passwordHash, role, fullName).
		Scan(&idStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.CreatedAt, &u.IsActive)
	if err != nil {
		return model.User{}, mapPGErr(err)
	}
	u.ID, _ = uuid.Parse(idStr)
	return u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	q := `SELECT id::text, email, password_hash, role, full_name, created_at, is_active FROM users WHERE email = $1`
	var idStr string
	var u model.User
	err := r.db.QueryRow(ctx, q, email).
		Scan(&idStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.CreatedAt, &u.IsActive)
	if err != nil {
		return model.User{}, err
	}
	u.ID, _ = uuid.Parse(idStr)
	return u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	q := `SELECT id::text, email, password_hash, role, full_name, created_at, is_active FROM users WHERE id = $1`
	var idStr string
	var u model.User
	err := r.db.QueryRow(ctx, q, id).
		Scan(&idStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.CreatedAt, &u.IsActive)
	if err != nil {
		return model.User{}, err
	}
	u.ID, _ = uuid.Parse(idStr)
	return u, nil
}

func (r *Repository) ListUsers(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	countQ := `SELECT COUNT(*) FROM users`
	var total int
	if err := r.db.QueryRow(ctx, countQ).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT id::text, email, role, full_name, created_at, is_active
	      FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := make([]model.User, 0)
	for rows.Next() {
		var idStr string
		var u model.User
		if err := rows.Scan(&idStr, &u.Email, &u.Role, &u.FullName, &u.CreatedAt, &u.IsActive); err != nil {
			return nil, 0, err
		}
		u.ID, _ = uuid.Parse(idStr)
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *Repository) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	q := `UPDATE users SET is_active = $2 WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id, isActive)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) error {
	q := `UPDATE users SET role = $2 WHERE id = $1`
	ct, err := r.db.Exec(ctx, q, id, role)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) GetLoanStats(ctx context.Context) (model.LoanStats, error) {
	q := `SELECT
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE status = 'pending') AS pending,
		COUNT(*) FILTER (WHERE status = 'approved') AS approved,
		COUNT(*) FILTER (WHERE status = 'rejected') AS rejected,
		COALESCE(SUM(amount), 0) AS total_amount
		FROM loan_applications`
	var s model.LoanStats
	err := r.db.QueryRow(ctx, q).Scan(&s.Total, &s.Pending, &s.Approved, &s.Rejected, &s.TotalAmount)
	return s, err
}

func (r *Repository) CancelLoanApplication(ctx context.Context, loanID, applicantID uuid.UUID) error {
	q := `UPDATE loan_applications SET status = 'rejected', updated_at = now()
	      WHERE id = $1 AND applicant_id = $2 AND status = 'pending'`
	ct, err := r.db.Exec(ctx, q, loanID, applicantID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	q := `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(ctx, q, userID, tokenHash, expiresAt)
	return err
}

func (r *Repository) GetUserByRefreshTokenHash(ctx context.Context, tokenHash string) (model.User, error) {
	q := `SELECT u.id::text, u.email, u.password_hash, u.role, u.full_name, u.created_at, u.is_active
	      FROM refresh_tokens rt
	      JOIN users u ON u.id = rt.user_id
	      WHERE rt.token_hash = $1
	      AND rt.revoked_at IS NULL
	      AND rt.expires_at > now()
	      AND u.is_active = true`
	var idStr string
	var u model.User
	err := r.db.QueryRow(ctx, q, tokenHash).
		Scan(&idStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.CreatedAt, &u.IsActive)
	if err != nil {
		return model.User{}, err
	}
	u.ID, _ = uuid.Parse(idStr)
	return u, nil
}

func (r *Repository) GetActiveRefreshToken(ctx context.Context, tokenHash string) (model.User, uuid.UUID, error) {
	q := `SELECT u.id::text, u.email, u.password_hash, u.role, u.full_name, u.created_at, u.is_active, rt.id::text
	      FROM refresh_tokens rt
	      JOIN users u ON u.id = rt.user_id
	      WHERE rt.token_hash = $1
	      AND rt.revoked_at IS NULL
	      AND rt.expires_at > now()
	      AND u.is_active = true`
	var userIDStr, tokenIDStr string
	var u model.User
	err := r.db.QueryRow(ctx, q, tokenHash).
		Scan(&userIDStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &u.CreatedAt, &u.IsActive, &tokenIDStr)
	if err != nil {
		return model.User{}, uuid.Nil, err
	}
	u.ID, _ = uuid.Parse(userIDStr)
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		return model.User{}, uuid.Nil, err
	}
	return u, tokenID, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	q := `UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`
	_, err := r.db.Exec(ctx, q, tokenHash)
	return err
}

func (r *Repository) RevokeRefreshTokenByID(ctx context.Context, tokenID uuid.UUID) error {
	q := `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`
	ct, err := r.db.Exec(ctx, q, tokenID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) RevokeRefreshTokenForUser(ctx context.Context, userID uuid.UUID, tokenHash string) error {
	q := `UPDATE refresh_tokens
	      SET revoked_at = now()
	      WHERE token_hash = $1 AND user_id = $2 AND revoked_at IS NULL`
	ct, err := r.db.Exec(ctx, q, tokenHash, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) CreateLoanApplication(ctx context.Context, applicantID uuid.UUID, amount float64, termMonths int, purpose string) (model.LoanApplication, error) {
	q := `INSERT INTO loan_applications (applicant_id, amount, term_months, purpose)
	      VALUES ($1, $2, $3, $4)
	      RETURNING id::text, applicant_id::text, amount, term_months, purpose, status, created_at, updated_at`
	var idStr, applicantStr string
	var loan model.LoanApplication
	err := r.db.QueryRow(ctx, q, applicantID, amount, termMonths, purpose).
		Scan(&idStr, &applicantStr, &loan.Amount, &loan.TermMonths, &loan.Purpose, &loan.Status, &loan.CreatedAt, &loan.UpdatedAt)
	if err != nil {
		return model.LoanApplication{}, err
	}
	loan.ID, _ = uuid.Parse(idStr)
	loan.ApplicantID, _ = uuid.Parse(applicantStr)
	return loan, nil
}

func (r *Repository) ListOwnLoanApplications(ctx context.Context, applicantID uuid.UUID, limit, offset int) ([]model.LoanApplication, int, error) {
	countQ := `SELECT COUNT(*) FROM loan_applications WHERE applicant_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQ, applicantID).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT id::text, applicant_id::text, amount, term_months, purpose, status, created_at, updated_at
	      FROM loan_applications
	      WHERE applicant_id = $1
	      ORDER BY created_at DESC
	      LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, applicantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.LoanApplication, 0)
	for rows.Next() {
		var idStr, applicantStr string
		var loan model.LoanApplication
		if err := rows.Scan(&idStr, &applicantStr, &loan.Amount, &loan.TermMonths, &loan.Purpose, &loan.Status, &loan.CreatedAt, &loan.UpdatedAt); err != nil {
			return nil, 0, err
		}
		loan.ID, _ = uuid.Parse(idStr)
		loan.ApplicantID, _ = uuid.Parse(applicantStr)
		items = append(items, loan)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetOwnLoanApplicationByID(ctx context.Context, loanID, applicantID uuid.UUID) (model.LoanApplication, error) {
	q := `SELECT id::text, applicant_id::text, amount, term_months, purpose, status, created_at, updated_at
	      FROM loan_applications WHERE id = $1 AND applicant_id = $2`
	var idStr, applicantStr string
	var loan model.LoanApplication
	err := r.db.QueryRow(ctx, q, loanID, applicantID).
		Scan(&idStr, &applicantStr, &loan.Amount, &loan.TermMonths, &loan.Purpose, &loan.Status, &loan.CreatedAt, &loan.UpdatedAt)
	if err != nil {
		return model.LoanApplication{}, err
	}
	loan.ID, _ = uuid.Parse(idStr)
	loan.ApplicantID, _ = uuid.Parse(applicantStr)
	return loan, nil
}

func (r *Repository) GetLoanApplicationByID(ctx context.Context, loanID uuid.UUID) (model.LoanApplication, error) {
	q := `SELECT id::text, applicant_id::text, amount, term_months, purpose, status, created_at, updated_at
	      FROM loan_applications WHERE id = $1`
	var idStr, applicantStr string
	var loan model.LoanApplication
	err := r.db.QueryRow(ctx, q, loanID).
		Scan(&idStr, &applicantStr, &loan.Amount, &loan.TermMonths, &loan.Purpose, &loan.Status, &loan.CreatedAt, &loan.UpdatedAt)
	if err != nil {
		return model.LoanApplication{}, err
	}
	loan.ID, _ = uuid.Parse(idStr)
	loan.ApplicantID, _ = uuid.Parse(applicantStr)
	return loan, nil
}

func (r *Repository) ListAllLoanApplications(ctx context.Context, status *string, limit, offset int) ([]model.LoanApplication, int, error) {
	var total int
	if status == nil || *status == "" {
		if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM loan_applications`).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err := r.db.Query(ctx, `SELECT id::text, applicant_id::text, amount, term_months, purpose, status, created_at, updated_at
			FROM loan_applications ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		items := make([]model.LoanApplication, 0)
		for rows.Next() {
			var idStr, applicantStr string
			var loan model.LoanApplication
			if err := rows.Scan(&idStr, &applicantStr, &loan.Amount, &loan.TermMonths, &loan.Purpose, &loan.Status, &loan.CreatedAt, &loan.UpdatedAt); err != nil {
				return nil, 0, err
			}
			loan.ID, _ = uuid.Parse(idStr)
			loan.ApplicantID, _ = uuid.Parse(applicantStr)
			items = append(items, loan)
		}
		return items, total, rows.Err()
	}

	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM loan_applications WHERE status = $1`, *status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `SELECT id::text, applicant_id::text, amount, term_months, purpose, status, created_at, updated_at
		FROM loan_applications WHERE status = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, *status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.LoanApplication, 0)
	for rows.Next() {
		var idStr, applicantStr string
		var loan model.LoanApplication
		if err := rows.Scan(&idStr, &applicantStr, &loan.Amount, &loan.TermMonths, &loan.Purpose, &loan.Status, &loan.CreatedAt, &loan.UpdatedAt); err != nil {
			return nil, 0, err
		}
		loan.ID, _ = uuid.Parse(idStr)
		loan.ApplicantID, _ = uuid.Parse(applicantStr)
		items = append(items, loan)
	}
	return items, total, rows.Err()
}

func (r *Repository) CreateLoanDecisionAndUpdateStatus(ctx context.Context, loanID, managerID uuid.UUID, decision, comment string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	updateQ := `UPDATE loan_applications
	            SET status = CASE WHEN $2 = 'approved' THEN 'approved' ELSE 'rejected' END,
	                updated_at = now()
	            WHERE id = $1`
	ct, err := tx.Exec(ctx, updateQ, loanID, decision)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	insertQ := `INSERT INTO loan_decisions (application_id, manager_id, decision, comment)
	            VALUES ($1, $2, $3, $4)`
	if _, err := tx.Exec(ctx, insertQ, loanID, managerID, decision, comment); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) IsUserActive(ctx context.Context, id uuid.UUID) (bool, error) {
	var isActive bool
	err := r.db.QueryRow(ctx, `SELECT is_active FROM users WHERE id = $1`, id).Scan(&isActive)
	if err != nil {
		return false, err
	}
	return isActive, nil
}

func (r *Repository) InsertAuditLog(ctx context.Context, entry model.AuditEntry) error {
	q := `INSERT INTO audit_logs (actor_id, action, resource, resource_id, ip_address, user_agent, details)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)`
	var raw []byte
	if entry.Details != nil {
		b, err := json.Marshal(entry.Details)
		if err != nil {
			return err
		}
		raw = b
	}
	_, err := r.db.Exec(ctx, q, entry.ActorID, entry.Action, entry.Resource, entry.ResourceID, entry.IPAddress, entry.UserAgent, raw)
	return err
}
