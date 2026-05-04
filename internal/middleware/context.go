package middleware

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const userKey ctxKey = "auth_user"

type AuthUser struct {
	ID   uuid.UUID
	Role string
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(userKey).(AuthUser)
	return u, ok
}
