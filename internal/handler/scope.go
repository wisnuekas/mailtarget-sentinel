package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
)

func companyFilter(c *fiber.Ctx, cfg *config.Config, store *redisstore.Store) *int32 {
	return service.ResolveCompanyScope(c.Context(), parseOptionalInt32Query(c, "company_id"), cfg, store)
}

func parseOptionalInt32Query(c *fiber.Ctx, key string) *int32 {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return nil
	}
	v := int32(id)
	return &v
}

func parseOptionalTimeQuery(c *fiber.Ctx, key string) *time.Time {
	raw := c.Query(key)
	if raw == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &t
}
