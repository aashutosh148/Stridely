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

		// Log incoming request
		slog.Info("→ incoming request",
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
			"query", c.Context().QueryArgs().String(),
		)

		// Log request body for POST/PUT/PATCH (first 500 chars)
		if c.Method() == "POST" || c.Method() == "PUT" || c.Method() == "PATCH" {
			body := c.Body()
			if len(body) > 0 {
				bodyStr := string(body)
				if len(bodyStr) > 500 {
					bodyStr = bodyStr[:500] + "... (truncated)"
				}
				slog.Debug("request body", "body", bodyStr)
			}
		}

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

		// Log response body for errors (first 500 chars)
		if status >= 400 {
			respBody := c.Response().Body()
			if len(respBody) > 0 {
				respStr := string(respBody)
				if len(respStr) > 500 {
					respStr = respStr[:500] + "... (truncated)"
				}
				attrs = append(attrs, "response_body", respStr)
			}
		}

		if status >= 500 {
			slog.Error("← request failed", attrs...)
		} else if status >= 400 {
			slog.Warn("← client error", attrs...)
		} else {
			slog.Info("← request completed", attrs...)
		}

		return err
	}
}
