package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/models"
	"github.com/aashutosh148/Stridely/pacer-api/services"
	"github.com/aashutosh148/Stridely/pacer-api/utils"
)

type AuthHandler struct {
	db       *db.Postgres
	strava   *services.StravaClient
	analysis *services.AnalysisService
}

func NewAuthHandler(dbConn *db.Postgres, stravaClient *services.StravaClient, analysisService *services.AnalysisService) *AuthHandler {
	return &AuthHandler{
		db:       dbConn,
		strava:   stravaClient,
		analysis: analysisService,
	}
}

// StravaLogin redirects to Strava OAuth page
func (h *AuthHandler) StravaLogin(c *fiber.Ctx) error {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	if clientID == "" {
		return c.Status(500).JSON(fiber.Map{"error": "STRAVA_CLIENT_ID not configured"})
	}

	// Build OAuth URL
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:3001"
	}
	redirectURI := fmt.Sprintf("%s/api/v1/auth/strava/callback", apiURL)

	authURL := fmt.Sprintf(
		"https://www.strava.com/oauth/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=read,activity:read_all",
		clientID, redirectURI,
	)

	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

// StravaCallback handles OAuth callback from Strava
func (h *AuthHandler) StravaCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(400).JSON(fiber.Map{"error": "missing authorization code"})
	}

	// Exchange code for tokens
	tokenResp, err := h.exchangeStravaCode(c.Context(), code)
	if err != nil {
		slog.Error("strava token exchange failed", "error", err)
		return c.Status(500).JSON(fiber.Map{"error": "token exchange failed"})
	}

	// Upsert user
	user, isNewUser, err := h.upsertUser(c.Context(), tokenResp)
	if err != nil {
		slog.Error("user upsert failed", "error", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to create/update user"})
	}

	// Store OAuth tokens (encrypted)
	if err := h.storeOAuthTokens(c.Context(), user.ID, tokenResp); err != nil {
		slog.Error("failed to store oauth tokens", "error", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to store tokens"})
	}

	// Generate JWT
	jwt, err := utils.SignToken(user.ID.String())
	if err != nil {
		slog.Error("jwt generation failed", "error", err)
		return c.Status(500).JSON(fiber.Map{"error": "failed to generate token"})
	}

	// Kick off background sync if new user
	if isNewUser {
		go func() {
			ctx := context.Background()
			if err := h.analysis.SyncStravaHistory(ctx, user.ID, h.strava, 180); err != nil {
				slog.Error("strava history sync failed", "user_id", user.ID, "error", err)
			}
		}()
		slog.Info("initiated strava history sync", "user_id", user.ID)
	}

	// Redirect to frontend with JWT token
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	redirectURL := fmt.Sprintf("%s/dashboard?token=%s", frontendURL, jwt)
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

// exchangeStravaCode exchanges authorization code for access token
func (h *AuthHandler) exchangeStravaCode(ctx context.Context, code string) (*services.StravaTokenResponse, error) {
	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("strava credentials not configured")
	}

	// Call Strava token endpoint
	url := fmt.Sprintf(
		"https://www.strava.com/oauth/token?client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code",
		clientID, clientSecret, code,
	)

	// Use fiber.AcquireAgent() for HTTP request
	agent := fiber.Post(url)
	agent.Set("Content-Type", "application/json")
	
	statusCode, body, errs := agent.Bytes()
	if len(errs) > 0 {
		return nil, fmt.Errorf("token request failed: %w", errs[0])
	}

	if statusCode != 200 {
		return nil, fmt.Errorf("token request failed with status %d", statusCode)
	}

	var tokenResp services.StravaTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &tokenResp, nil
}

// upsertUser creates or updates user based on Strava athlete data
func (h *AuthHandler) upsertUser(ctx context.Context, tokenResp *services.StravaTokenResponse) (*models.User, bool, error) {
	athleteID := strconv.FormatInt(tokenResp.Athlete.ID, 10)

	// Check if user exists
	var existingUserID uuid.UUID
	err := h.db.Pool.QueryRow(ctx, `
		SELECT id FROM users WHERE strava_athlete_id = $1
	`, athleteID).Scan(&existingUserID)

	isNewUser := err == sql.ErrNoRows

	if isNewUser {
		// Create new user
		var newUserID uuid.UUID
		err = h.db.Pool.QueryRow(ctx, `
			INSERT INTO users (
				email, runner_tier, subscription_tier, strava_athlete_id,
				preferred_language, notification_prefs, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, now(), now())
			RETURNING id
		`,
			fmt.Sprintf("%d@strava.user", tokenResp.Athlete.ID), // Temporary email
			models.RunnerTierRecreational,
			models.SubscriptionTierFree,
			athleteID,
			"en",
			`{"push": true, "quiet_hours": ["22:00", "06:00"]}`,
		).Scan(&newUserID)

		if err != nil {
			return nil, false, fmt.Errorf("create user: %w", err)
		}

		slog.Info("created new user", "user_id", newUserID, "strava_athlete_id", athleteID)

		return &models.User{
			ID:                newUserID,
			Email:             fmt.Sprintf("%d@strava.user", tokenResp.Athlete.ID),
			RunnerTier:        models.RunnerTierRecreational,
			SubscriptionTier:  models.SubscriptionTierFree,
			StravaAthleteID:   sql.NullString{String: athleteID, Valid: true},
			PreferredLanguage: "en",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}, true, nil
	}

	// Update existing user
	_, err = h.db.Pool.Exec(ctx, `
		UPDATE users SET updated_at = now() WHERE id = $1
	`, existingUserID)

	if err != nil {
		return nil, false, fmt.Errorf("update user: %w", err)
	}

	slog.Info("user login", "user_id", existingUserID, "strava_athlete_id", athleteID)

	// Fetch full user record
	var user models.User
	err = h.db.Pool.QueryRow(ctx, `
		SELECT id, email, runner_tier, subscription_tier, strava_athlete_id, created_at, updated_at
		FROM users WHERE id = $1
	`, existingUserID).Scan(
		&user.ID, &user.Email, &user.RunnerTier, &user.SubscriptionTier,
		&user.StravaAthleteID, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, false, fmt.Errorf("fetch user: %w", err)
	}

	return &user, false, nil
}

// storeOAuthTokens encrypts and stores OAuth tokens
func (h *AuthHandler) storeOAuthTokens(ctx context.Context, userID uuid.UUID, tokenResp *services.StravaTokenResponse) error {
	// Encrypt tokens
	accessTokenEnc, err := utils.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	refreshTokenEnc, err := utils.Encrypt(tokenResp.RefreshToken)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	// Store in database
	expiresAt := time.Unix(tokenResp.ExpiresAt, 0)
	_, err = h.db.Pool.Exec(ctx, `
		INSERT INTO oauth_tokens (
			user_id, provider, access_token_enc, refresh_token_enc,
			expires_at, scope, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, now(), now())
		ON CONFLICT (user_id, provider)
		DO UPDATE SET
			access_token_enc = EXCLUDED.access_token_enc,
			refresh_token_enc = EXCLUDED.refresh_token_enc,
			expires_at = EXCLUDED.expires_at,
			scope = EXCLUDED.scope,
			updated_at = now()
	`,
		userID, models.OAuthProviderStrava,
		accessTokenEnc, refreshTokenEnc,
		expiresAt, "read,activity:read_all",
	)

	if err != nil {
		return fmt.Errorf("store oauth tokens: %w", err)
	}

	slog.Info("oauth tokens stored", "user_id", userID, "provider", "strava")
	return nil
}

// Me returns current user info
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var user models.User
	err := h.db.Pool.QueryRow(c.Context(), `
		SELECT id, email, runner_tier, subscription_tier, goal_time_s,
		       target_race_date, strava_athlete_id, garmin_user_id, created_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Email, &user.RunnerTier, &user.SubscriptionTier,
		&user.GoalTimeS, &user.TargetRaceDate, &user.StravaAthleteID,
		&user.GarminUserID, &user.CreatedAt,
	)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	// Check connection status
	var stravaConnected, garminConnected bool
	stravaConnected = user.StravaAthleteID.Valid
	garminConnected = user.GarminUserID.Valid

	return c.JSON(fiber.Map{
		"user":              user,
		"strava_connected":  stravaConnected,
		"garmin_connected":  garminConnected,
	})
}
