package models

import (
	"time"

	"github.com/google/uuid"
)

type OAuthProvider string

const (
	OAuthProviderStrava OAuthProvider = "strava"
	OAuthProviderGarmin OAuthProvider = "garmin"
)

type OAuthToken struct {
	ID               uuid.UUID     `json:"id"`
	UserID           uuid.UUID     `json:"user_id"`
	Provider         OAuthProvider `json:"provider"`
	AccessTokenEnc   []byte        `json:"-"` // Never serialize
	RefreshTokenEnc  []byte        `json:"-"` // Never serialize
	ExpiresAt        time.Time     `json:"expires_at"`
	Scope            string        `json:"scope,omitempty"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}
