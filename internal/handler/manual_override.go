package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type ManualOverrideHandler struct {
	cfg         *config.Config
	alerts      *sqlite.AlertRepository
	subAccounts *postgres.SubAccountRepository
}

func NewManualOverrideHandler(
	cfg *config.Config,
	alerts *sqlite.AlertRepository,
	subAccounts *postgres.SubAccountRepository,
) *ManualOverrideHandler {
	return &ManualOverrideHandler{cfg: cfg, alerts: alerts, subAccounts: subAccounts}
}

type manualOverrideRequest struct {
	SubAccountID int32  `json:"sub_account_id"`
	Action       string `json:"action"`
}

type manualOverrideResponse struct {
	SubAccountID int32  `json:"sub_account_id"`
	Status       string `json:"status"`
	Action       string `json:"action"`
}

func (h *ManualOverrideHandler) Execute(c *fiber.Ctx) error {
	if h.cfg.AdminToken != "" {
		auth := c.Get("Authorization")
		expected := "Bearer " + h.cfg.AdminToken
		if auth != expected {
			return response.Unauthorized(c, "invalid admin token")
		}
	}

	var req manualOverrideRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	if req.SubAccountID == 0 {
		return response.BadRequest(c, "sub_account_id is required")
	}

	var alertStatus string
	switch req.Action {
	case "suspend":
		alertStatus = sqlite.StatusSuspended
	case "resume":
		alertStatus = sqlite.StatusResolved
	default:
		return response.BadRequest(c, "action must be 'suspend' or 'resume'")
	}

	status, err := applySubAccountStatus(c.Context(), h.subAccounts, req.SubAccountID, req.Action)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	_ = h.alerts.UpdateStatusBySubAccount(c.Context(), req.SubAccountID, alertStatus)

	return response.OK(c, manualOverrideResponse{
		SubAccountID: req.SubAccountID,
		Status:       status,
		Action:       req.Action,
	})
}
