package middleware

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/yourname/pacer-api/utils"
)

// RecoveryMiddleware recovers from panics and logs to Sentry
func RecoveryMiddleware() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			// Log panic
			slog.Error("panic recovered",
				"error", e,
				"path", c.Path(),
				"method", c.Method(),
			)

		},
	})
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	var appErr *utils.AppError
	if errors.As(err, &appErr) {
		if appErr.Err != nil && appErr.Code >= 500 {
			slog.Error("app error", "path", c.Path(), "err", appErr.Err)
		}
		return c.Status(appErr.Code).JSON(fiber.Map{"error": appErr.Message})
	}

	if e, ok := err.(*fiber.Error); ok {
		return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
	}

	slog.Error("unhandled error", "path", c.Path(), "err", err)
	return c.Status(500).JSON(fiber.Map{"error": "internal server error"})
}
