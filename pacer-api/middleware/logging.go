package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LoggingMiddleware logs all HTTP requests with structured logging
func LoggingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Log after processing
		duration := time.Since(start)
		status := c.Response().StatusCode()

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.IP(),
		}

		// Add userID if authenticated
		if userID, ok := c.Locals("userID").(string); ok {
			attrs = append(attrs, "user_id", userID)
		}

		if status >= 500 {
			slog.Error("request failed", attrs...)
		} else if status >= 400 {
			slog.Warn("client error", attrs...)
		} else {
			slog.Info("request", attrs...)
		}

		return err
	}
}
