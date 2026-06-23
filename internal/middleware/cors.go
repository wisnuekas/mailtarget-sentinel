package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func DashboardCORS(origins string) fiber.Handler {
	allowed := strings.Split(origins, ",")
	for i, o := range allowed {
		allowed[i] = strings.TrimSpace(o)
	}

	return cors.New(cors.Config{
		AllowOrigins: strings.Join(allowed, ","),
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	})
}
