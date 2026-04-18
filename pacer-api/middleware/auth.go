package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourname/pacer-api/utils"
)

// AuthMiddleware verifies JWT tokens and injects userID into context
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.VerifyToken(tokenStr)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		// Inject userID into context for downstream handlers
		c.Locals("userID", claims.UserID)
		return c.Next()
	}
}
