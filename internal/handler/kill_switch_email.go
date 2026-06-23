package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/worker"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type KillSwitchEmailHandler struct {
	cfg      *config.Config
	detector *worker.Detector
}

func NewKillSwitchEmailHandler(cfg *config.Config, detector *worker.Detector) *KillSwitchEmailHandler {
	return &KillSwitchEmailHandler{cfg: cfg, detector: detector}
}

type killSwitchEmailRequest struct {
	SubAccountID int32  `json:"sub_account_id"`
	AlertID      string `json:"alert_id"`
	Window       string `json:"window"`
}

func (h *KillSwitchEmailHandler) requireAdmin(c *fiber.Ctx) error {
	if h.cfg.AdminToken == "" {
		return nil
	}
	if c.Get("Authorization") != "Bearer "+h.cfg.AdminToken {
		return response.Unauthorized(c, "invalid admin token")
	}
	return nil
}

func (h *KillSwitchEmailHandler) Send(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	var req killSwitchEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	var result *worker.KillSwitchEmailResult
	var err error

	if req.AlertID != "" {
		result, err = h.detector.ResendKillSwitchEmailByAlertID(c.Context(), req.AlertID)
	} else if req.SubAccountID != 0 {
		windowStr := req.Window
		if windowStr == "" {
			windowStr = "5m"
		}
		window, parseErr := parseWindow(windowStr)
		if parseErr != nil {
			return response.BadRequest(c, parseErr.Error())
		}
		result, err = h.detector.ResendKillSwitchEmail(c.Context(), req.SubAccountID, window)
	} else {
		return response.BadRequest(c, "sub_account_id or alert_id is required")
	}

	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, result)
}
