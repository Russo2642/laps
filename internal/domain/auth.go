package domain

import (
	"time"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Session struct {
	ID           string    `json:"id"`
	UserID       int64     `json:"user_id"`
	RefreshToken string    `json:"refresh_token"`
	UserAgent    string    `json:"user_agent"`
	IP           string    `json:"ip"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegisterRequest struct {
	FirstName        string   `json:"first_name" binding:"required"`
	LastName         string   `json:"last_name" binding:"required"`
	MiddleName       string   `json:"middle_name"`
	Email            string   `json:"email" binding:"required,email"`
	Phone            string   `json:"phone" binding:"required"`
	Password         string   `json:"password" binding:"required,min=6"`
	Role             UserRole `json:"role" binding:"required,oneof=client specialist"`
	SpecializationID *int64   `json:"specialization_id"`
}

type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
