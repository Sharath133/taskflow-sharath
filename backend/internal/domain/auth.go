package domain

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthResponse is returned after successful login or registration.
type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// Claims is the JWT payload for TaskFlow access tokens (HS256).
// Standard times use exp and iat via RegisteredClaims; identity uses user_id and email.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// ParsedUserID returns the user_id claim as a UUID.
func (c *Claims) ParsedUserID() (uuid.UUID, error) {
	if c.UserID == "" {
		return uuid.Nil, errors.New("missing user_id")
	}
	id, err := uuid.Parse(c.UserID)
	if err != nil {
		return uuid.Nil, errors.New("invalid user_id")
	}
	return id, nil
}
