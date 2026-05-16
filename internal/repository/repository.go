package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"bank-loan-mvp/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrConflict = errors.New("conflict")

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return pgx.ErrNoRows
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrConflict
	}
	return err
}

func noRows(n int64) error {
	if n == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, _ = time.Parse("2006-01-02 15:04:05", s)
	}
	return t
}

func scanUser(row *sql.Row) (model.User, error) {
	var u model.User
	var idStr, createdAt string
	var isActive int
	err := row.Scan(&idStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &createdAt, &isActive)
	if err != nil {
		return model.User{}, mapErr(err)
	}
	u.ID, _ = uuid.Parse(idStr)
	u.CreatedAt = parseTime(createdAt)
	u.IsActive = isActive != 0
	return u, nil
}

func scanLoan(row *sql.Row) (model.LoanApplication, error) {
	var loan model.LoanApplication
	var idStr, applicantStr, createdAt, updatedAt string
	var termMonths int64
	err := row.Scan(&idStr, &applicantStr, &loan.Amount, &termMonths, &loan.Purpose, &loan.Status, &createdAt, &updatedAt)
	if err != nil {
		return model.LoanApplication{}, mapErr(err)
	}
	loan.ID, _ = uuid.Parse(idStr)
	loan.ApplicantID, _ = uuid.Parse(applicantStr)
	loan.TermMonths = int(termMonths)
	loan.CreatedAt = parseTime(createdAt)
	loan.UpdatedAt = parseTime(updatedAt)
	return loan, nil
}

func scanLoanRows(rows *sql.Rows) ([]model.LoanApplication, error) {
	items := make([]model.LoanApplication, 0)
	for rows.Next() {
		var loan model.LoanApplication
		var idStr, applicantStr, createdAt, updatedAt string
		var termMonths int64
		if err := rows.Scan(&idStr, &applicantStr, &loan.Amount, &termMonths, &loan.Purpose, &loan.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		loan.ID, _ = uuid.Parse(idStr)
		loan.ApplicantID, _ = uuid.Parse(applicantStr)
		loan.TermMonths = int(termMonths)
		loan.CreatedAt = parseTime(createdAt)
		loan.UpdatedAt = parseTime(updatedAt)
		items = append(items, loan)
	}
	return items, rows.Err()
}

func (r *Repository) CreateUser(ctx context.Context, email, passwordHash, role, fullName string) (model.User, error) {
	id := uuid.New().String()
	q := `INSERT INTO users (id, email, password_hash, role, full_name) VALUES (?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, q, id, email, passwordHash, role, fullName); err != nil {
		return model.User{}, mapErr(err)
	}
	return r.GetUserByEmail(ctx, email)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	q := `SELECT id, email, password_hash, role, full_name, created_at, is_active FROM users WHERE email = ?`
	return scanUser(r.db.QueryRowContext(ctx, q, email))
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (model.User, error) {
	q := `SELECT id, email, password_hash, role, full_name, created_at, is_active FROM users WHERE id = ?`
	return scanUser(r.db.QueryRowContext(ctx, q, id.String()))
}

func (r *Repository) ListUsers(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT id, email, role, full_name, created_at, is_active FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		var idStr, createdAt string
		var isActive int
		if err := rows.Scan(&idStr, &u.Email, &u.Role, &u.FullName, &createdAt, &isActive); err != nil {
			return nil, 0, err
		}
		u.ID, _ = uuid.Parse(idStr)
		u.CreatedAt = parseTime(createdAt)
		u.IsActive = isActive != 0
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *Repository) UpdateUserStatus(ctx context.Context, id uuid.UUID, isActive bool) error {
	active := 0
	if isActive {
		active = 1
	}
	res, err := r.db.ExecContext(ctx, `UPDATE users SET is_active = ? WHERE id = ?`, active, id.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	return noRows(n)
}

func (r *Repository) UpdateUserRole(ctx context.Context, id uuid.UUID, role string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, role, id.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	return noRows(n)
}

func (r *Repository) GetLoanStats(ctx context.Context) (model.LoanStats, error) {
	q := `SELECT
		COUNT(*),
		SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END),
		COALESCE(SUM(amount), 0)
		FROM loan_applications`
	var s model.LoanStats
	err := r.db.QueryRowContext(ctx, q).Scan(&s.Total, &s.Pending, &s.Approved, &s.Rejected, &s.TotalAmount)
	return s, err
}

func (r *Repository) CancelLoanApplication(ctx context.Context, loanID, applicantID uuid.UUID) error {
	q := `UPDATE loan_applications SET status = 'rejected', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	      WHERE id = ? AND applicant_id = ? AND status = 'pending'`
	res, err := r.db.ExecContext(ctx, q, loanID.String(), applicantID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	return noRows(n)
}

func (r *Repository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	id := uuid.New().String()
	q := `INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, q, id, userID.String(), tokenHash, expiresAt.UTC().Format(time.RFC3339))
	return err
}

func (r *Repository) GetUserByRefreshTokenHash(ctx context.Context, tokenHash string) (model.User, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.role, u.full_name, u.created_at, u.is_active
	      FROM refresh_tokens rt
	      JOIN users u ON u.id = rt.user_id
	      WHERE rt.token_hash = ?
	      AND rt.revoked_at IS NULL
	      AND rt.expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	      AND u.is_active = 1`
	return scanUser(r.db.QueryRowContext(ctx, q, tokenHash))
}

func (r *Repository) GetActiveRefreshToken(ctx context.Context, tokenHash string) (model.User, uuid.UUID, error) {
	q := `SELECT u.id, u.email, u.password_hash, u.role, u.full_name, u.created_at, u.is_active, rt.id
	      FROM refresh_tokens rt
	      JOIN users u ON u.id = rt.user_id
	      WHERE rt.token_hash = ?
	      AND rt.revoked_at IS NULL
	      AND rt.expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	      AND u.is_active = 1`
	var u model.User
	var userIDStr, tokenIDStr, createdAt string
	var isActive int
	err := r.db.QueryRowContext(ctx, q, tokenHash).
		Scan(&userIDStr, &u.Email, &u.PasswordHash, &u.Role, &u.FullName, &createdAt, &isActive, &tokenIDStr)
	if err != nil {
		return model.User{}, uuid.Nil, mapErr(err)
	}
	u.ID, _ = uuid.Parse(userIDStr)
	u.CreatedAt = parseTime(createdAt)
	u.IsActive = isActive != 0
	tokenID, _ := uuid.Parse(tokenIDStr)
	return u, tokenID, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	q := `UPDATE refresh_tokens SET revoked_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE token_hash = ? AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, tokenHash)
	return err
}

func (r *Repository) RevokeRefreshTokenByID(ctx context.Context, tokenID uuid.UUID) error {
	q := `UPDATE refresh_tokens SET revoked_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = ? AND revoked_at IS NULL`
	res, err := r.db.ExecContext(ctx, q, tokenID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	return noRows(n)
}

func (r *Repository) RevokeRefreshTokenForUser(ctx context.Context, userID uuid.UUID, tokenHash string) error {
	q := `UPDATE refresh_tokens SET revoked_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	      WHERE token_hash = ? AND user_id = ? AND revoked_at IS NULL`
	res, err := r.db.ExecContext(ctx, q, tokenHash, userID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	return noRows(n)
}

func (r *Repository) CreateLoanApplication(ctx context.Context, applicantID uuid.UUID, amount float64, termMonths int, purpose string) (model.LoanApplication, error) {
	id := uuid.New()
	q := `INSERT INTO loan_applications (id, applicant_id, amount, term_months, purpose) VALUES (?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, q, id.String(), applicantID.String(), amount, termMonths, purpose); err != nil {
		return model.LoanApplication{}, err
	}
	return r.GetLoanApplicationByID(ctx, id)
}

func (r *Repository) ListOwnLoanApplications(ctx context.Context, applicantID uuid.UUID, limit, offset int) ([]model.LoanApplication, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM loan_applications WHERE applicant_id = ?`, applicantID.String()).Scan(&total); err != nil {
		return nil, 0, err
	}
	q := `SELECT id, applicant_id, amount, term_months, purpose, status, created_at, updated_at
	      FROM loan_applications WHERE applicant_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, q, applicantID.String(), limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanLoanRows(rows)
	return items, total, err
}

func (r *Repository) GetOwnLoanApplicationByID(ctx context.Context, loanID, applicantID uuid.UUID) (model.LoanApplication, error) {
	q := `SELECT id, applicant_id, amount, term_months, purpose, status, created_at, updated_at
	      FROM loan_applications WHERE id = ? AND applicant_id = ?`
	return scanLoan(r.db.QueryRowContext(ctx, q, loanID.String(), applicantID.String()))
}

func (r *Repository) GetLoanApplicationByID(ctx context.Context, loanID uuid.UUID) (model.LoanApplication, error) {
	q := `SELECT id, applicant_id, amount, term_months, purpose, status, created_at, updated_at
	      FROM loan_applications WHERE id = ?`
	return scanLoan(r.db.QueryRowContext(ctx, q, loanID.String()))
}

func (r *Repository) ListAllLoanApplications(ctx context.Context, status *string, limit, offset int) ([]model.LoanApplication, int, error) {
	var total int
	if status == nil || *status == "" {
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM loan_applications`).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err := r.db.QueryContext(ctx,
			`SELECT id, applicant_id, amount, term_months, purpose, status, created_at, updated_at
			 FROM loan_applications ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		items, err := scanLoanRows(rows)
		return items, total, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM loan_applications WHERE status = ?`, *status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, applicant_id, amount, term_months, purpose, status, created_at, updated_at
		 FROM loan_applications WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, *status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanLoanRows(rows)
	return items, total, err
}

func (r *Repository) CreateLoanDecisionAndUpdateStatus(ctx context.Context, loanID, managerID uuid.UUID, decision, comment string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	updateQ := `UPDATE loan_applications
	            SET status = CASE WHEN ? = 'approved' THEN 'approved' ELSE 'rejected' END,
	                updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
	            WHERE id = ?`
	res, err := tx.ExecContext(ctx, updateQ, decision, loanID.String())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return pgx.ErrNoRows
	}

	id := uuid.New().String()
	insertQ := `INSERT INTO loan_decisions (id, application_id, manager_id, decision, comment) VALUES (?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, insertQ, id, loanID.String(), managerID.String(), decision, comment); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) IsUserActive(ctx context.Context, id uuid.UUID) (bool, error) {
	var isActive int
	err := r.db.QueryRowContext(ctx, `SELECT is_active FROM users WHERE id = ?`, id.String()).Scan(&isActive)
	if err != nil {
		return false, err
	}
	return isActive != 0, nil
}

func (r *Repository) InsertAuditLog(ctx context.Context, entry model.AuditEntry) error {
	q := `INSERT INTO audit_logs (actor_id, action, resource, resource_id, ip_address, user_agent, details)
	      VALUES (?, ?, ?, ?, ?, ?, ?)`
	var raw []byte
	if entry.Details != nil {
		b, err := json.Marshal(entry.Details)
		if err != nil {
			return err
		}
		raw = b
	}
	var actorID *string
	if entry.ActorID != nil {
		s := entry.ActorID.String()
		actorID = &s
	}
	_, err := r.db.ExecContext(ctx, q, actorID, entry.Action, entry.Resource, entry.ResourceID, entry.IPAddress, entry.UserAgent, raw)
	return err
}
