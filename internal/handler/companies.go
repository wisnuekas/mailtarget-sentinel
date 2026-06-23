package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/pkg/response"
)

type CompaniesHandler struct {
	cfg       *config.Config
	companies *postgres.CompanyRepository
	events    *chrepo.EventRepository
	store     *redisstore.Store
	enricher  *service.RiskEnricher
}

func NewCompaniesHandler(
	cfg *config.Config,
	companies *postgres.CompanyRepository,
	events *chrepo.EventRepository,
	store *redisstore.Store,
	enricher *service.RiskEnricher,
) *CompaniesHandler {
	return &CompaniesHandler{
		cfg:       cfg,
		companies: companies,
		events:    events,
		store:     store,
		enricher:  enricher,
	}
}

type companyRow struct {
	ID        int32   `json:"id"`
	Name      string  `json:"name"`
	Active    bool    `json:"active"`
	AtRisk    bool    `json:"at_risk,omitempty"`
	MaxBounce float64 `json:"max_bounce_rate_pct,omitempty"`
}

func (h *CompaniesHandler) List(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	search := c.Query("search")
	atRiskOnly := c.Query("at_risk") == "true"

	list, total, err := h.companies.List(c.Context(), postgres.CompanyListFilter{
		CompanyID: companyFilter(c, h.cfg, h.store),
		Search:    search,
		Page:      page,
		Size:      size,
	})
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	riskByCompany := map[int32]float64{}
	if atRiskOnly || c.Query("include_risk") == "true" {
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
		for _, s := range summary {
			riskByCompany[s.CompanyID] = s.MaxBounceRatePct
		}
	}

	rows := make([]companyRow, 0, len(list))
	for _, co := range list {
		row := companyRow{ID: co.ID, Name: co.Name, Active: co.Active}
		if bounce, ok := riskByCompany[co.ID]; ok {
			row.AtRisk = true
			row.MaxBounce = bounce
		}
		if atRiskOnly && !row.AtRisk {
			continue
		}
		rows = append(rows, row)
	}

	if atRiskOnly {
		total = len(rows)
	}

	return response.OK(c, fiber.Map{
		"page":      page,
		"size":      size,
		"count":     total,
		"companies": rows,
	})
}
