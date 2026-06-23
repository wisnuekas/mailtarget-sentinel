package handler

import (
	"strconv"
	"time"

	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
	"github.com/gofiber/fiber/v2"
)

type AlertsHandler struct {
	alerts *sqlite.AlertRepository
}

func NewAlertsHandler(alerts *sqlite.AlertRepository) *AlertsHandler {
	return &AlertsHandler{alerts: alerts}
}

type alertsListResponse struct {
	Alerts []sqlite.Alert `json:"alerts"`
	Total  int            `json:"total"`
	Page   int            `json:"page"`
	Limit  int            `json:"limit"`
}

func (h *AlertsHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	filter := sqlite.AlertFilter{
		CompanyID:    parseOptionalInt32Query(c, "company_id"),
		SubAccountID: parseOptionalInt32Query(c, "sub_account_id"),
		Status:       c.Query("status"),
		From:         parseOptionalTimeQuery(c, "from"),
		To:           parseOptionalTimeQuery(c, "to"),
		Page:         page,
		Limit:        limit,
	}

	alerts, total, err := h.alerts.List(c.Context(), filter)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	if alerts == nil {
		alerts = []sqlite.Alert{}
	}

	return response.OK(c, alertsListResponse{
		Alerts: alerts,
		Total:  total,
		Page:   page,
		Limit:  limit,
	})
}

func (h *AlertsHandler) Get(c *fiber.Ctx) error {
	alert, err := h.alerts.GetByAlertID(c.Context(), c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "alert not found")
	}
	return response.OK(c, alert)
}

func (h *AlertsHandler) Overview(c *fiber.Ctx) error {
	recent, err := h.alerts.Recent(c.Context(), 10)
	if err != nil {
		return response.InternalError(c, err.Error())
	}
	stats, err := h.alerts.Stats(c.Context())
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	if recent == nil {
		recent = []sqlite.Alert{}
	}

	return response.OK(c, fiber.Map{
		"recent_alerts": recent,
		"stats_by_status": stats,
		"generated_at": time.Now().UTC(),
	})
}
