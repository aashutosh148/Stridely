package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type bucket struct {
	count      int
	windowEnds time.Time
}

var (
	chatBuckets = map[string]*bucket{}
	chatMu      sync.Mutex
)

func ChatRateLimit(maxPerMinute int) fiber.Handler {
	if maxPerMinute <= 0 {
		maxPerMinute = 100
	}

	return func(c *fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)
		if userID == "" {
			return c.Next()
		}

		now := time.Now()
		chatMu.Lock()
		defer chatMu.Unlock()

		b, ok := chatBuckets[userID]
		if !ok || now.After(b.windowEnds) {
			b = &bucket{count: 0, windowEnds: now.Add(time.Minute)}
			chatBuckets[userID] = b
		}

		b.count++
		if b.count > maxPerMinute {
			retryAfter := int(time.Until(b.windowEnds).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "rate limited, try in 1min",
			})
		}

		return c.Next()
	}
}
