package model

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
	FullName string `json:"full_name" validate:"required,min=2,max=120"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=20,max=2048"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,min=20,max=2048"`
}

type CreateLoanRequest struct {
	Amount     float64 `json:"amount" validate:"required,gt=0,lte=10000000"`
	TermMonths int     `json:"term_months" validate:"required,min=1,max=360"`
	Purpose    string  `json:"purpose" validate:"required,min=3,max=500"`
}

type DecideLoanRequest struct {
	Decision string `json:"decision" validate:"required,oneof=approved rejected"`
	Comment  string `json:"comment" validate:"max=500"`
}

type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=client manager admin"`
}

type LoanStats struct {
	Total    int     `json:"total"`
	Pending  int     `json:"pending"`
	Approved int     `json:"approved"`
	Rejected int     `json:"rejected"`
	TotalAmount float64 `json:"total_amount"`
}

type AuthTokensResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	FullName  string `json:"full_name"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
}
