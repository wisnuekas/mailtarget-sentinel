package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AMPCORS sets headers required for AMP action-xhr form submissions.
func AMPCORS() fiber.Handler {
	return func(c *fiber.Ctx) error {
		origin := c.Get("Origin")
		if origin == "" {
			origin = "https://mail.google.com"
		}

		c.Set("Access-Control-Allow-Origin", origin)
		c.Set("AMP-Access-Control-Allow-Source-Origin", extractSourceOrigin(c))
		c.Set("Access-Control-Expose-Headers", "AMP-Access-Control-Allow-Source-Origin")
		c.Set("Access-Control-Allow-Headers", "Content-Type")
		c.Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

func extractSourceOrigin(c *fiber.Ctx) string {
	if v := c.Get("AMP-Access-Control-Allow-Source-Origin"); v != "" {
		return v
	}
	host := c.Hostname()
	if host == "" {
		return "https://mailtarget.co"
	}
	if strings.HasPrefix(host, "localhost") {
		return "http://" + host
	}
	return "https://" + host
}
