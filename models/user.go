package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	Password     string         `json:"-"`
	Name         string         `json:"name"`
	OIDCSubject  string         `gorm:"uniqueIndex" json:"-"`     // OIDC sub claim
	OIDCIssuer   string         `json:"-"`                        // OIDC issuer URL
	AuthProvider string         `gorm:"default:'local'" json:"-"` // 'local' or 'oidc'
	CurrentToken string         `json:"-"`                        // Current active session token
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Diagrams     []Diagram      `gorm:"foreignKey:UserID" json:"diagrams,omitempty"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UserSignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

type UserResponse struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
