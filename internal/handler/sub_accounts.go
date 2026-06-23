package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/alert"
	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type SubAccountsHandler struct {
	cfg          *config.Config
	subAccounts  *postgres.SubAccountRepository
	companies    *postgres.CompanyRepository
	store        *redisstore.Store
	transmission *mailtarget.TransmissionClient
	events       *chrepo.EventRepository
}

func NewSubAccountsHandler(
	cfg *config.Config,
	subAccounts *postgres.SubAccountRepository,
	companies *postgres.CompanyRepository,
	store *redisstore.Store,
	transmission *mailtarget.TransmissionClient,
	events *chrepo.EventRepository,
) *SubAccountsHandler {
	return &SubAccountsHandler{
		cfg:          cfg,
		subAccounts:  subAccounts,
		companies:    companies,
		store:        store,
		transmission: transmission,
		events:       events,
	}
}

type subAccountRow struct {
	ID            int32                     `json:"id"`
	Name          string                    `json:"name"`
	Status        string                    `json:"status"`
	CreatedAt     int64                     `json:"created_at"`
	CompanyID     int32                     `json:"company_id,omitempty"`
	CompanyName   string                    `json:"company_name,omitempty"`
	IPPoolName    string                    `json:"ip_pool_name,omitempty"`
	Metrics       *chrepo.SubAccountMetrics `json:"metrics,omitempty"`
	Domains       []postgres.DomainLite     `json:"domains,omitempty"`
}

type subAccountListResponse struct {
	Page        int             `json:"page"`
	Size        int             `json:"size"`
	Count       int             `json:"count"`
	Window      string          `json:"window"`
	SubAccounts []subAccountRow `json:"sub_accounts"`
}

func (h *SubAccountsHandler) requireAdmin(c *fiber.Ctx) error {
	if h.cfg.AdminToken == "" {
		return nil
	}
	if c.Get("Authorization") != "Bearer "+h.cfg.AdminToken {
		return response.Unauthorized(c, "invalid admin token")
	}
	return nil
}

func (h *SubAccountsHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	search := c.Query("search")
	status := c.Query("status")
	windowStr := c.Query("window", "5m")

	window, err := parseWindow(windowStr)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	filter := postgres.SubAccountListFilter{
		CompanyID: companyFilter(c, h.cfg, h.store),
		Search:    search,
		Status:    status,
		Page:      page,
		Size:      size,
	}

	if c.Query("all") != "true" {
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
		if len(summary) == 0 {
			return response.OK(c, subAccountListResponse{
				Page:        page,
				Size:        size,
				Count:       0,
				Window:      windowStr,
				SubAccounts: []subAccountRow{},
			})
		}
		companyIDs := make([]int32, 0, len(summary))
		for _, s := range summary {
			companyIDs = append(companyIDs, s.CompanyID)
		}
		filter.CompanyID = nil
		filter.CompanyIDs = companyIDs
	}

	list, count, err := h.subAccounts.List(c.Context(), filter)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	metricsByID := map[int32]chrepo.SubAccountMetrics{}
	metrics, err := h.events.GetMetrics(c.Context(), companyFilter(c, h.cfg, h.store), window, nil)
	if err == nil {
		for _, m := range metrics {
			metricsByID[m.SubAccountID] = m
		}
	}

	rows := make([]subAccountRow, 0, len(list))
	for _, sa := range list {
		row := subAccountRow{
			ID:          sa.ID,
			Name:        sa.Name,
			Status:      sa.Status,
			CreatedAt:   sa.CreatedAt,
			CompanyID:   sa.CompanyID,
			CompanyName: sa.CompanyName,
			IPPoolName:  sa.IPPoolName,
		}
		if m, ok := metricsByID[sa.ID]; ok {
			mCopy := m
			row.Metrics = &mCopy
		}
		rows = append(rows, row)
	}

	return response.OK(c, subAccountListResponse{
		Page:        page,
		Size:        size,
		Count:       count,
		Window:      windowStr,
		SubAccounts: rows,
	})
}

func (h *SubAccountsHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 32)
	if err != nil || id <= 0 {
		return response.BadRequest(c, "invalid sub-account id")
	}

	windowStr := c.Query("window", "5m")
	window, err := parseWindow(windowStr)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	sa, err := h.subAccounts.GetByID(c.Context(), int32(id))
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	row := subAccountRow{
		ID:          sa.ID,
		Name:        sa.Name,
		Status:      sa.Status,
		CreatedAt:   sa.CreatedAt,
		CompanyID:   sa.CompanyID,
		CompanyName: sa.CompanyName,
		IPPoolName:  sa.IPPoolName,
		Domains:     sa.Domains,
	}

	subID := int32(id)
	metrics, err := h.events.GetMetrics(c.Context(), companyFilter(c, h.cfg, h.store), window, &subID)
	if err == nil && len(metrics) > 0 {
		mCopy := metrics[0]
		row.Metrics = &mCopy
	}

	return response.OK(c, fiber.Map{
		"window":      windowStr,
		"sub_account": row,
	})
}

type warningEmailRequest struct {
	SubAccountID int32  `json:"sub_account_id"`
	Window       string `json:"window"`
}

type warningEmailResponse struct {
	SubAccountID   int32  `json:"sub_account_id"`
	TransmissionID string `json:"transmission_id"`
	Subject        string `json:"subject"`
}

func (h *SubAccountsHandler) SendWarning(c *fiber.Ctx) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	var req warningEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}
	if req.SubAccountID == 0 {
		return response.BadRequest(c, "sub_account_id is required")
	}

	windowStr := req.Window
	if windowStr == "" {
		windowStr = "5m"
	}
	window, err := parseWindow(windowStr)
	if err != nil {
		return response.BadRequest(c, err.Error())
	}

	sa, err := h.subAccounts.GetByID(c.Context(), req.SubAccountID)
	if err != nil {
		return response.InternalError(c, fmt.Sprintf("fetch sub-account: %v", err))
	}

	var metrics chrepo.SubAccountMetrics
	subID := req.SubAccountID
	chMetrics, err := h.events.GetMetrics(c.Context(), companyFilter(c, h.cfg, h.store), window, &subID)
	if err == nil && len(chMetrics) > 0 {
		metrics = chMetrics[0]
	} else {
		metrics = chrepo.SubAccountMetrics{
			CompanyID:    sa.CompanyID,
			SubAccountID: req.SubAccountID,
		}
	}

	email := alert.BuildWarningEmail(alert.WarningEmailInput{
		SubAccountID:   req.SubAccountID,
		SubAccountName: sa.Name,
		CompanyID:      metrics.CompanyID,
		Sent:           metrics.Sent,
		Bounced:        metrics.Bounced,
		SpamBounced:    metrics.SpamBounced,
		BounceRate:     metrics.BounceRatePct,
		SpamRate:       metrics.SpamRatePct,
	})

	companyID := metrics.CompanyID
	if companyID == 0 {
		companyID = sa.CompanyID
	}

	form := mailtarget.TransmissionForm{
		Subject:  email.Subject,
		From:     mailtarget.Address{Email: h.cfg.Alert.FromEmail, Name: h.cfg.Alert.FromName},
		To:       service.ResolveEmailRecipients(c.Context(), h.companies, h.cfg, companyID),
		BodyText: email.BodyText,
		BodyHTML: email.BodyHTML,
		Metadata: map[string]string{
			"sentinel_type":  "warning",
			"sub_account_id": fmt.Sprintf("%d", req.SubAccountID),
			"company_id":     fmt.Sprintf("%d", metrics.CompanyID),
			"sent_at":        time.Now().UTC().Format(time.RFC3339),
		},
		OptionsAttributes: &mailtarget.OptionsAttributes{
			ClickTracking: false,
			OpenTracking:  false,
			Transactional: true,
		},
	}

	result, err := h.transmission.Send(c.Context(), form)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.OK(c, warningEmailResponse{
		SubAccountID:   req.SubAccountID,
		TransmissionID: result.TransmissionID,
		Subject:        email.Subject,
	})
}
