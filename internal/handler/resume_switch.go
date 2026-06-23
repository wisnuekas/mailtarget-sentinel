package handler

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type ResumeSwitchHandler struct {
	cfg         *config.Config
	store       *redisstore.Store
	alerts      *sqlite.AlertRepository
	subAccounts *postgres.SubAccountRepository
}

func NewResumeSwitchHandler(
	cfg *config.Config,
	store *redisstore.Store,
	alerts *sqlite.AlertRepository,
	subAccounts *postgres.SubAccountRepository,
) *ResumeSwitchHandler {
	return &ResumeSwitchHandler{cfg: cfg, store: store, alerts: alerts, subAccounts: subAccounts}
}

type resumeSwitchResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	SubAccountID int32  `json:"sub_account_id"`
	Status       string `json:"status"`
}

func (h *ResumeSwitchHandler) Execute(c *fiber.Ctx) error {
	token, subAccountID, err := parseResumeSwitchInput(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	payload, err := h.store.ConsumeResumeToken(c.Context(), token)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}

	if payload.SubAccountID != subAccountID {
		return response.Unauthorized(c, "token does not match sub_account_id")
	}

	status, err := applySubAccountStatus(c.Context(), h.subAccounts, subAccountID, "resume")
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	if payload.AlertID != "" {
		_ = h.alerts.UpdateStatus(c.Context(), payload.AlertID, sqlite.StatusResolved, nil)
	} else {
		_ = h.alerts.UpdateStatusBySubAccount(c.Context(), subAccountID, sqlite.StatusResolved)
	}

	resp := resumeSwitchResponse{
		Success:      true,
		Message:      "Sub-account resumed via email",
		SubAccountID: subAccountID,
		Status:       status,
	}

	if c.Get("AMP-Same-Origin") != "" || c.Get("Origin") == "https://mail.google.com" {
		return c.JSON(resp)
	}

	return response.OKMessage(c, resp.Message, resp)
}

func parseResumeSwitchInput(c *fiber.Ctx) (string, int32, error) {
	token := c.FormValue("token")
	subAccountRaw := c.FormValue("sub_account_id")

	if token == "" {
		var body struct {
			Token        string `json:"token"`
			SubAccountID int32  `json:"sub_account_id"`
		}
		if err := c.BodyParser(&body); err == nil && body.Token != "" {
			token = body.Token
			subAccountRaw = strconv.Itoa(int(body.SubAccountID))
		}
	}

	if token == "" {
		token = c.Query("token")
	}
	if subAccountRaw == "" {
		subAccountRaw = c.Query("sub_account_id")
	}

	if token == "" || subAccountRaw == "" {
		return "", 0, fmt.Errorf("token and sub_account_id are required")
	}

	id, err := strconv.ParseInt(subAccountRaw, 10, 32)
	if err != nil {
		return "", 0, fmt.Errorf("invalid sub_account_id")
	}

	return token, int32(id), nil
}
