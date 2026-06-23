package handler

import (
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
)

type SettingsHandler struct {
	cfg   *config.Config
	store *redisstore.Store
}

func NewSettingsHandler(cfg *config.Config, store *redisstore.Store) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, store: store}
}

type settingsResponse struct {
	MinVolume              uint64  `json:"min_volume"`
	BounceRateThresholdPct float64 `json:"bounce_rate_threshold_pct"`
	SpamRateThresholdPct   float64 `json:"spam_rate_threshold_pct"`
	AlertCooldownMinutes   int     `json:"alert_cooldown_minutes"`
	CompanyID              int32   `json:"company_id"`
}

func toSettingsResponse(settings redisstore.Settings, cfg *config.Config) settingsResponse {
	return settingsResponse{
		MinVolume:              settings.MinVolume,
		BounceRateThresholdPct: settings.BounceRateThresholdPct,
		SpamRateThresholdPct:   settings.SpamRateThresholdPct,
		AlertCooldownMinutes:   settings.AlertCooldownMinutes,
		CompanyID:              service.EffectiveCompanyID(settings, cfg),
	}
}

func (h *SettingsHandler) Update(c *fiber.Ctx) error {
	var body settingsResponse
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	companyID := body.CompanyID
	partial := redisstore.Settings{
		MinVolume:              body.MinVolume,
		BounceRateThresholdPct: body.BounceRateThresholdPct,
		SpamRateThresholdPct:   body.SpamRateThresholdPct,
		AlertCooldownMinutes:   body.AlertCooldownMinutes,
		CompanyID:              &companyID,
	}

	settings, err := h.store.UpdateSettings(c.Context(), partial)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, toSettingsResponse(settings, h.cfg))
}

func (h *SettingsHandler) Get(c *fiber.Ctx) error {
	settings, err := h.store.GetSettings(c.Context())
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	return response.OK(c, toSettingsResponse(settings, h.cfg))
}
