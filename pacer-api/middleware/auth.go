package middleware

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/aashutosh148/Stridely/pacer-api/utils"
)

// AuthMiddleware verifies JWT tokens and injects userID into context
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		slog.Debug("🔒 auth middleware checking token", "path", c.Path())
		
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			slog.Warn("missing or invalid authorization header", "path", c.Path(), "header", header[:min(len(header), 20)])
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		slog.Debug("verifying token", "token_length", len(tokenStr))
		
		claims, err := utils.VerifyToken(tokenStr)
		if err != nil {
			slog.Error("token verification failed", "error", err, "path", c.Path())
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		slog.Debug("✅ token verified", "user_id", claims.UserID, "path", c.Path())
		
		// Inject userID into context for downstream handlers
		c.Locals("userID", claims.UserID)
		return c.Next()
	}
}
