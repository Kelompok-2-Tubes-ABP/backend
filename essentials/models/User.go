package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Email    string             `bson:"email" json:"email"`
	Password string             `bson:"password" json:"password"`
	Role     string             `bson:"role" json:"role"` // "admin" or "user"

	// Email verification
	IsEmailVerified   bool       `bson:"is_email_verified" json:"is_email_verified"`
	EmailVerifiedAt   *time.Time `bson:"email_verified_at,omitempty" json:"email_verified_at,omitempty"`
	VerificationToken string     `bson:"verification_token,omitempty" json:"verification_token,omitempty"`

	// Password reset
	ResetToken        string     `bson:"reset_token,omitempty" json:"reset_token,omitempty"`
	ResetTokenExpiry  *time.Time `bson:"reset_token_expiry,omitempty" json:"reset_token_expiry,omitempty"`
	PasswordChangedAt time.Time  `bson:"password_changed_at" json:"password_changed_at"`

	// Account status
	IsActive  bool      `bson:"is_active" json:"is_active"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// VerificationRequest represents an email verification request
type VerificationRequest struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Token     string             `bson:"token" json:"token"`
	Email     string             `bson:"email" json:"email"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	UsedAt    *time.Time         `bson:"used_at,omitempty" json:"used_at,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Token     string             `bson:"token" json:"token"`
	Email     string             `bson:"email" json:"email"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	UsedAt    *time.Time         `bson:"used_at,omitempty" json:"used_at,omitempty"`
	IPAddress string             `bson:"ip_address" json:"ip_address"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
