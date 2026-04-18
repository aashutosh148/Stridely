package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourname/pacer-api/db"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *db.Postgres
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(database *db.Postgres) *HealthHandler {
	return &HealthHandler{db: database}
}

// Check returns health status for Railway health checks
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	if h.db == nil || h.db.Pool == nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "degraded",
			"version": 1,
			"db":      "missing_pool",
		})
	}

	if err := h.db.Pool.Ping(c.Context()); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"status":  "degraded",
			"version": 1,
			"db":      "unhealthy",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"version": 1,
		"db":      "ok",
	})
}
