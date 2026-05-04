package audit

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"bank-loan-mvp/internal/model"
	"bank-loan-mvp/internal/repository"
)

type Logger struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Logger {
	return &Logger{repo: repo}
}

func (l *Logger) Log(ctx context.Context, entry model.AuditEntry) {
	entry.CreatedAt = time.Now().UTC()
	if err := l.repo.InsertAuditLog(ctx, entry); err != nil {
		log.Printf("audit_db_write_failed=%v", err)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		log.Printf("audit_marshal_failed=%v", err)
		return
	}
	log.Printf("%s", string(line))
}
