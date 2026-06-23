package handler

import (
	"fmt"
	"strconv"
	"time"

	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type MetricsHandler struct {
	cfg    *config.Config
	events *chrepo.EventRepository
	store  *redisstore.Store
}

func NewMetricsHandler(cfg *config.Config, events *chrepo.EventRepository, store *redisstore.Store) *MetricsHandler {
	return &MetricsHandler{cfg: cfg, events: events, store: store}
}

type metricsResponse struct {
	Window    string                    `json:"window"`
	CompanyID *int32                    `json:"company_id,omitempty"`
	Metrics   []chrepo.SubAccountMetrics `json:"metrics"`
}

func (h *MetricsHandler) Get(c *fiber.Ctx) error {
	window, err := parseWindow(c.Query("window", "5m"))
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	companyID := h.optionalCompanyID(c)
	var subAccountID *int32
	if raw := c.Query("sub_account_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return response.BadRequest(c, "invalid sub_account_id")
		}
		v := int32(id)
		subAccountID = &v
	}

	metrics, err := h.events.GetMetrics(c.Context(), companyID, window, subAccountID)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	if metrics == nil {
		metrics = []chrepo.SubAccountMetrics{}
	}

	return response.OK(c, metricsResponse{
		Window:    c.Query("window", "5m"),
		CompanyID: companyID,
		Metrics:   metrics,
	})
}

func (h *MetricsHandler) optionalCompanyID(c *fiber.Ctx) *int32 {
	return companyFilter(c, h.cfg, h.store)
}

func parseWindow(raw string) (time.Duration, error) {
	switch raw {
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	default:
		return 0, fmt.Errorf("window must be one of: 5m, 15m, 1h")
	}
}
