package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type AtRiskHandler struct {
	cfg      *config.Config
	events   *chrepo.EventRepository
	store    *redisstore.Store
	enricher *service.RiskEnricher
}

func NewAtRiskHandler(
	cfg *config.Config,
	events *chrepo.EventRepository,
	store *redisstore.Store,
	enricher *service.RiskEnricher,
) *AtRiskHandler {
	return &AtRiskHandler{cfg: cfg, events: events, store: store, enricher: enricher}
}

func (h *AtRiskHandler) List(c *fiber.Ctx) error {
	window, err := parseWindow(c.Query("window", "5m"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	settings, err := h.store.GetSettings(c.Context())
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	payload, err := h.enricher.BuildAtRisk(
		c.Context(),
		companyFilter(c, h.cfg, h.store),
		window,
		settings,
		c.Query("window", "5m"),
	)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, payload)
}

func (h *AtRiskHandler) Summary(c *fiber.Ctx) error {
	window, err := parseWindow(c.Query("window", "5m"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	settings, err := h.store.GetSettings(c.Context())
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	summary, err := h.events.GetAtRiskSummaryForCompany(
		c.Context(), companyFilter(c, h.cfg, h.store), window,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	if summary == nil {
		summary = []chrepo.CompanyRiskSummary{}
	}

	enriched := h.enricher.EnrichSummary(c.Context(), summary)

	return response.OK(c, fiber.Map{
		"window":  c.Query("window", "5m"),
		"summary": enriched,
		"count":   len(enriched),
	})
}
