package handler

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type KillSwitchHandler struct {
	cfg          *config.Config
	store        *redisstore.Store
	alerts       *sqlite.AlertRepository
	subAccounts  *postgres.SubAccountRepository
	companies    *postgres.CompanyRepository
	transmission *mailtarget.TransmissionClient
}

func NewKillSwitchHandler(
	cfg *config.Config,
	store *redisstore.Store,
	alerts *sqlite.AlertRepository,
	subAccounts *postgres.SubAccountRepository,
	companies *postgres.CompanyRepository,
	transmission *mailtarget.TransmissionClient,
) *KillSwitchHandler {
	return &KillSwitchHandler{
		cfg:          cfg,
		store:        store,
		alerts:       alerts,
		subAccounts:  subAccounts,
		companies:    companies,
		transmission: transmission,
	}
}

type killSwitchResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	SubAccountID int32  `json:"sub_account_id"`
	Status       string `json:"status"`
}

func (h *KillSwitchHandler) Execute(c *fiber.Ctx) error {
	token, subAccountID, err := parseKillSwitchInput(c)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	payload, err := h.store.ConsumeKillToken(c.Context(), token)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}

	if payload.SubAccountID != subAccountID {
		return response.Unauthorized(c, "token does not match sub_account_id")
	}

	status, err := applySubAccountStatus(c.Context(), h.subAccounts, subAccountID, "suspend")
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	settings, _ := h.store.GetSettings(c.Context())
	cooldown := time.Duration(settings.AlertCooldownMinutes) * time.Minute
	_ = h.store.ExtendAlertLock(c.Context(), subAccountID, cooldown)

	if err := h.alerts.UpdateStatus(c.Context(), payload.Alert.AlertID, sqlite.StatusSuspended, nil); err != nil {
		_ = h.alerts.UpdateStatusBySubAccount(c.Context(), subAccountID, sqlite.StatusSuspended)
	}

	companyID := payload.Alert.CompanyID
	if companyID == 0 {
		if sa, err := h.subAccounts.GetByID(c.Context(), subAccountID); err == nil {
			companyID = sa.CompanyID
		}
	}

	if err := service.SendResumeConfirmationEmail(
		c.Context(),
		h.cfg,
		h.companies,
		h.transmission,
		h.store,
		subAccountID,
		companyID,
		payload.Alert.AlertID,
	); err != nil {
		slog.Warn("kill-switch: resume confirmation email failed",
			"sub_account_id", subAccountID,
			"company_id", companyID,
			"error", err,
		)
	}

	resp := killSwitchResponse{
		Success:      true,
		Message:      "Sub-account suspended via kill switch",
		SubAccountID: subAccountID,
		Status:       status,
	}

	if c.Get("AMP-Same-Origin") != "" || c.Get("Origin") == "https://mail.google.com" {
		return c.JSON(resp)
	}

	return response.OKMessage(c, resp.Message, resp)
}

func parseKillSwitchInput(c *fiber.Ctx) (string, int32, error) {
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
