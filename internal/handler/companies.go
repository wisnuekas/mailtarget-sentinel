package handler

import (
	"strconv"
	"strings"

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
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}

	if c.Query("all") == "true" {
		return h.listAll(c, page, size)
	}
	return h.listAtRisk(c, page, size)
}

func (h *CompaniesHandler) listAtRisk(c *fiber.Ctx, page, size int) error {
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))

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
	companyIDs := make([]int32, 0, len(enriched))
	for _, s := range enriched {
		companyIDs = append(companyIDs, s.CompanyID)
	}

	compMap, _ := h.companies.GetByIDs(c.Context(), companyIDs)

	filtered := make([]companyRow, 0, len(enriched))
	for _, s := range enriched {
		name := s.CompanyName
		active := true
		if co, ok := compMap[s.CompanyID]; ok {
			if name == "" {
				name = co.Name
			}
			active = co.Active
		}
		if search != "" && !strings.Contains(strings.ToLower(name), search) {
			continue
		}
		filtered = append(filtered, companyRow{
			ID:        s.CompanyID,
			Name:      name,
			Active:    active,
			AtRisk:    true,
			MaxBounce: s.MaxBounceRatePct,
		})
	}

	total := len(filtered)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}

	return response.OK(c, fiber.Map{
		"page":      page,
		"size":      size,
		"count":     total,
		"companies": filtered[start:end],
	})
}

func (h *CompaniesHandler) listAll(c *fiber.Ctx, page, size int) error {
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
