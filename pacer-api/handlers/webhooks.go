package handlers

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourname/pacer-api/db"
	"github.com/yourname/pacer-api/models"
	"github.com/yourname/pacer-api/services"
)

type WebhookHandler struct {
	db       *db.Postgres
	redis    *db.Redis
	strava   *services.StravaClient
	analysis *services.AnalysisService
	events   *services.EventHub
}

func NewWebhookHandler(dbConn *db.Postgres, redisConn *db.Redis, stravaClient *services.StravaClient, analysisService *services.AnalysisService, events *services.EventHub) *WebhookHandler {
	return &WebhookHandler{
		db:       dbConn,
		redis:    redisConn,
		strava:   stravaClient,
		analysis: analysisService,
		events:   events,
	}
}

// StravaWebhookEvent represents the webhook payload from Strava
type StravaWebhookEvent struct {
	ObjectType     string `json:"object_type"` // "activity" or "athlete"
	ObjectID       int64  `json:"object_id"`   // Activity ID
	AspectType     string `json:"aspect_type"` // "create", "update", or "delete"
	OwnerID        int64  `json:"owner_id"`    // Athlete ID
	SubscriptionID int    `json:"subscription_id"`
	EventTime      int64  `json:"event_time"` // Unix timestamp
}

// StravaVerify handles Strava webhook verification challenge (GET request)
func (h *WebhookHandler) StravaVerify(c *fiber.Ctx) error {
	challenge := c.Query("hub.challenge")
	verifyToken := c.Query("hub.verify_token")

	expectedToken := os.Getenv("STRAVA_WEBHOOK_VERIFY_TOKEN")
	if expectedToken == "" {
		expectedToken = "pacer_webhook_verify" // Default for development
	}

	if verifyToken != expectedToken {
		slog.Warn("invalid webhook verify token", "received", verifyToken)
		return c.Status(403).JSON(fiber.Map{"error": "invalid verify token"})
	}

	slog.Info("strava webhook verification successful", "challenge", challenge)
	return c.JSON(fiber.Map{"hub.challenge": challenge})
}

// StravaWebhook handles Strava webhook events (POST request)
func (h *WebhookHandler) StravaWebhook(c *fiber.Ctx) error {
	var event StravaWebhookEvent
	if err := c.BodyParser(&event); err != nil {
		slog.Warn("invalid webhook payload", "error", err)
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	slog.Info("received strava webhook",
		"object_type", event.ObjectType,
		"aspect_type", event.AspectType,
		"object_id", event.ObjectID,
		"owner_id", event.OwnerID,
	)

	// IMPORTANT: Respond in < 2 seconds (Strava requirement)
	// Process the event asynchronously
	go h.processStravaEvent(event)

	return c.Status(200).JSON(fiber.Map{"status": "received"})
}

// processStravaEvent processes webhook event asynchronously
func (h *WebhookHandler) processStravaEvent(event StravaWebhookEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	eventID := strconv.FormatInt(event.EventTime, 10) + ":" + strconv.FormatInt(event.ObjectID, 10) + ":" + event.AspectType
	if h.isWebhookDuplicate(ctx, eventID) {
		slog.Info("webhook duplicate skipped", "event_id", eventID)
		return
	}

	// Only process activity events
	if event.ObjectType != "activity" {
		slog.Info("ignoring non-activity event", "object_type", event.ObjectType)
		return
	}

	// Only process create and update events
	if event.AspectType != "create" && event.AspectType != "update" {
		slog.Info("ignoring non-create/update event", "aspect_type", event.AspectType)
		return
	}

	// Look up user by Strava athlete ID
	athleteID := strconv.FormatInt(event.OwnerID, 10)
	var user models.User
	err := h.db.Pool.QueryRow(ctx, `
		SELECT id, email, runner_tier, subscription_tier, threshold_pace_s, max_hr, strava_athlete_id
		FROM users
		WHERE strava_athlete_id = $1
	`, athleteID).Scan(
		&user.ID, &user.Email, &user.RunnerTier, &user.SubscriptionTier,
		&user.ThresholdPaceS, &user.MaxHR, &user.StravaAthleteID,
	)

	if err == sql.ErrNoRows {
		slog.Warn("webhook: user not found for athlete", "athlete_id", athleteID)
		return
	}

	if err != nil {
		slog.Error("webhook: failed to lookup user", "athlete_id", athleteID, "error", err)
		return
	}

	// Fetch activity detail from Strava
	activity, err := h.strava.GetActivityDetail(ctx, user.ID.String(), event.ObjectID)
	if err != nil {
		slog.Error("webhook: failed to fetch activity detail",
			"activity_id", event.ObjectID,
			"user_id", user.ID,
			"error", err,
		)
		return
	}

	// Process and store activity
	if err := h.analysis.ProcessNewActivity(ctx, &user, activity); err != nil {
		slog.Error("webhook: failed to process activity",
			"activity_id", event.ObjectID,
			"user_id", user.ID,
			"error", err,
		)
		return
	}

	slog.Info("webhook: activity processed successfully",
		"activity_id", event.ObjectID,
		"user_id", user.ID,
		"aspect_type", event.AspectType,
	)

	if h.events != nil {
		preview := "Your workout has been synced. Debrief is ready."
		distanceKM := activity.Distance / 1000.0
		if distanceKM > 0 {
			preview = "Synced run: " + formatDistance(distanceKM) + " km. Debrief is ready."
		}
		h.events.Publish(user.ID, "post_workout", map[string]any{
			"activity_id":      event.ObjectID,
			"aspect_type":      event.AspectType,
			"workout_type":     activity.Type,
			"distance_m":       activity.Distance,
			"duration_s":       activity.MovingTime,
			"debrief_preview":  preview,
			"received_at":      time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func formatDistance(km float64) string {
	return strconv.FormatFloat(km, 'f', 1, 64)
}

func (h *WebhookHandler) isWebhookDuplicate(ctx context.Context, eventID string) bool {
	if h.redis == nil || h.redis.Client == nil {
		return false
	}

	key := "processed_webhook:" + eventID
	set, err := h.redis.Client.SetNX(ctx, key, "1", 24*time.Hour).Result()
	if err != nil {
		slog.Warn("webhook dedupe check failed", "event_id", eventID, "error", err)
		return false
	}
	return !set
}
