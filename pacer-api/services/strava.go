package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/aashutosh148/Stridely/pacer-api/db"
	"github.com/aashutosh148/Stridely/pacer-api/models"
	"github.com/aashutosh148/Stridely/pacer-api/utils"
)

type StravaClient struct {
	db       *db.Postgres
	http     *http.Client
	baseURL  string
	authBase string
}

type StravaAthlete struct {
	ID            int64  `json:"id"`
	Username      string `json:"username"`
	Firstname     string `json:"firstname"`
	Lastname      string `json:"lastname"`
	Bio           string `json:"bio"`
	City          string `json:"city"`
	State         string `json:"state"`
	Profile       string `json:"profile"`
	ProfileMedium string `json:"profile_medium"`
}

type StravaTokenResponse struct {
	TokenType    string        `json:"token_type"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresAt    int64         `json:"expires_at"`
	Athlete      StravaAthlete `json:"athlete"`
}

type StravaActivity struct {
	ID int64 `json:"id"`
}

type StravaSplit struct {
	Distance            float64 `json:"distance"`
	MovingTime          int     `json:"moving_time"`
	AverageHeartrate    float64 `json:"average_heartrate"`
	ElevationDifference float64 `json:"elevation_difference"`
}

type StravaDetailedActivity struct {
	ID                 int64         `json:"id"`
	Type               string        `json:"type"`
	SportType          string        `json:"sport_type"`
	StartDate          string        `json:"start_date"`
	Distance           float64       `json:"distance"`
	MovingTime         int           `json:"moving_time"`
	TotalElevationGain float64       `json:"total_elevation_gain"`
	AverageHeartrate   float64       `json:"average_heartrate"`
	MaxHeartrate       float64       `json:"max_heartrate"`
	HasHeartrate       bool          `json:"has_heartrate"`
	WorkoutType        int           `json:"workout_type"`
	GearID             string        `json:"gear_id"`
	SplitsMetric       []StravaSplit `json:"splits_metric"`
}

func NewStravaClient(database *db.Postgres) *StravaClient {
	return &StravaClient{
		db:       database,
		http:     &http.Client{Timeout: 20 * time.Second},
		baseURL:  "https://www.strava.com/api/v3",
		authBase: "https://www.strava.com/oauth/token",
	}
}

func (s *StravaClient) GetActivities(ctx context.Context, userID string, afterTimestamp int64, perPage int) ([]StravaActivity, error) {
	if perPage <= 0 {
		perPage = 30
	}

	token, err := s.getAccessToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/athlete/activities?after=%d&per_page=%d", s.baseURL, afterTimestamp, perPage),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("strava activities request failed with status %d", resp.StatusCode)
	}

	var out []StravaActivity
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *StravaClient) GetActivityDetail(ctx context.Context, userID string, activityID int64) (*StravaDetailedActivity, error) {
	token, err := s.getAccessToken(ctx, userID)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/activities/%d?include_all_efforts=true", s.baseURL, activityID),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("strava activity detail request failed with status %d", resp.StatusCode)
	}

	var out StravaDetailedActivity
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *StravaClient) getAccessToken(ctx context.Context, userID string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", err
	}

	var token models.OAuthToken
	err = s.db.Pool.QueryRow(ctx, `
    SELECT id, user_id, provider, access_token_enc, refresh_token_enc, expires_at, scope, created_at, updated_at
    FROM oauth_tokens
    WHERE user_id = $1 AND provider = $2
  `, uid, models.OAuthProviderStrava).Scan(
		&token.ID,
		&token.UserID,
		&token.Provider,
		&token.AccessTokenEnc,
		&token.RefreshTokenEnc,
		&token.ExpiresAt,
		&token.Scope,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		return "", err
	}

	if time.Until(token.ExpiresAt) < 5*time.Minute {
		refreshed, err := s.refreshToken(ctx, uid, &token)
		if err != nil {
			return "", err
		}
		token = *refreshed
	}

	access, err := utils.Decrypt(token.AccessTokenEnc)
	if err != nil {
		return "", err
	}
	return access, nil
}

func (s *StravaClient) refreshToken(ctx context.Context, userID uuid.UUID, token *models.OAuthToken) (*models.OAuthToken, error) {
	refreshToken, err := utils.Decrypt(token.RefreshTokenEnc)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.authBase, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Set("client_id", getenv("STRAVA_CLIENT_ID", ""))
	q.Set("client_secret", getenv("STRAVA_CLIENT_SECRET", ""))
	q.Set("grant_type", "refresh_token")
	q.Set("refresh_token", refreshToken)
	req.URL.RawQuery = q.Encode()

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("strava refresh failed with status %d", resp.StatusCode)
	}

	var tr StravaTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}

	accessEnc, err := utils.Encrypt(tr.AccessToken)
	if err != nil {
		return nil, err
	}
	refreshEnc, err := utils.Encrypt(tr.RefreshToken)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Pool.Exec(ctx, `
    UPDATE oauth_tokens
    SET access_token_enc = $1,
        refresh_token_enc = $2,
        expires_at = to_timestamp($3),
        updated_at = NOW()
    WHERE user_id = $4 AND provider = $5
  `, accessEnc, refreshEnc, tr.ExpiresAt, userID, models.OAuthProviderStrava)
	if err != nil {
		return nil, err
	}

	token.AccessTokenEnc = accessEnc
	token.RefreshTokenEnc = refreshEnc
	token.ExpiresAt = time.Unix(tr.ExpiresAt, 0)
	return token, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
