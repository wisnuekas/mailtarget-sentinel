package service

import (
	"context"
	"time"

	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
)

type RiskEnricher struct {
	events    *chrepo.EventRepository
	companies *postgres.CompanyRepository
	subAccts  *postgres.SubAccountRepository
	domains   *postgres.DomainRepository
}

func NewRiskEnricher(
	events *chrepo.EventRepository,
	companies *postgres.CompanyRepository,
	subAccts *postgres.SubAccountRepository,
	domains *postgres.DomainRepository,
) *RiskEnricher {
	return &RiskEnricher{
		events:    events,
		companies: companies,
		subAccts:  subAccts,
		domains:   domains,
	}
}

type AtRiskSubAccount struct {
	chrepo.SubAccountMetrics
	SubAccountName string `json:"sub_account_name,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
	Status         string `json:"status,omitempty"`
}

type AtRiskCompanySummary struct {
	chrepo.CompanyRiskSummary
	CompanyName string `json:"company_name,omitempty"`
}

type AtRiskDomain struct {
	chrepo.DomainMetrics
	DomainID    int32  `json:"domain_id,omitempty"`
	IsSending   bool   `json:"is_sending,omitempty"`
	IsBlocked   bool   `json:"is_blocked,omitempty"`
}

type AtRiskSendingIP struct {
	chrepo.SendingIPMetrics
}

type AtRiskPayload struct {
	Window      string                 `json:"window"`
	Items       []AtRiskSubAccount     `json:"items"`
	Summary     []AtRiskCompanySummary `json:"summary"`
	Domains     []AtRiskDomain         `json:"domains"`
	SendingIPs  []AtRiskSendingIP      `json:"sending_ips"`
}

func (e *RiskEnricher) BuildAtRisk(
	ctx context.Context,
	companyID *int32,
	window time.Duration,
	settings redisstore.Settings,
	windowLabel string,
) (*AtRiskPayload, error) {
	items, err := e.events.DetectAnomalies(
		ctx, companyID, window,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		return nil, err
	}

	summary, err := e.events.GetAtRiskSummaryForCompany(
		ctx, companyID, window,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		return nil, err
	}

	domains, err := e.events.DetectAtRiskDomains(
		ctx, companyID, window,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		return nil, err
	}

	sendingIPs, err := e.events.DetectAtRiskSendingIPs(
		ctx, companyID, window,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []chrepo.SubAccountMetrics{}
	}
	if summary == nil {
		summary = []chrepo.CompanyRiskSummary{}
	}
	if domains == nil {
		domains = []chrepo.DomainMetrics{}
	}
	if sendingIPs == nil {
		sendingIPs = []chrepo.SendingIPMetrics{}
	}

	subIDs := make([]int32, 0, len(items))
	companyIDs := make([]int32, 0, len(summary))
	domainNames := make([]string, 0, len(domains))

	for _, m := range items {
		subIDs = append(subIDs, m.SubAccountID)
	}
	for _, s := range summary {
		companyIDs = append(companyIDs, s.CompanyID)
	}
	for _, d := range domains {
		domainNames = append(domainNames, d.SendingDomain)
	}

	subMap, _ := e.subAccts.GetByIDs(ctx, subIDs)
	compMap, _ := e.companies.GetByIDs(ctx, companyIDs)
	domMap, _ := e.domains.FindByDomainNames(ctx, domainNames, companyID)

	enrichedItems := make([]AtRiskSubAccount, 0, len(items))
	for _, m := range items {
		row := AtRiskSubAccount{SubAccountMetrics: m}
		if sa, ok := subMap[m.SubAccountID]; ok {
			row.SubAccountName = sa.Name
			row.Status = sa.Status
			if sa.CompanyName != "" {
				row.CompanyName = sa.CompanyName
			}
		}
		if row.CompanyName == "" {
			if c, ok := compMap[m.CompanyID]; ok {
				row.CompanyName = c.Name
			}
		}
		enrichedItems = append(enrichedItems, row)
	}

	enrichedSummary := make([]AtRiskCompanySummary, 0, len(summary))
	for _, s := range summary {
		row := AtRiskCompanySummary{CompanyRiskSummary: s}
		if c, ok := compMap[s.CompanyID]; ok {
			row.CompanyName = c.Name
		}
		enrichedSummary = append(enrichedSummary, row)
	}

	enrichedDomains := make([]AtRiskDomain, 0, len(domains))
	for _, d := range domains {
		row := AtRiskDomain{DomainMetrics: d}
		if rec, ok := domMap[d.SendingDomain]; ok {
			row.DomainID = rec.ID
			row.IsSending = rec.IsSending
			row.IsBlocked = rec.IsBlocked
		}
		enrichedDomains = append(enrichedDomains, row)
	}

	enrichedSendingIPs := make([]AtRiskSendingIP, 0, len(sendingIPs))
	for _, ip := range sendingIPs {
		enrichedSendingIPs = append(enrichedSendingIPs, AtRiskSendingIP{SendingIPMetrics: ip})
	}

	return &AtRiskPayload{
		Window:     windowLabel,
		Items:      enrichedItems,
		Summary:    enrichedSummary,
		Domains:    enrichedDomains,
		SendingIPs: enrichedSendingIPs,
	}, nil
}

func (e *RiskEnricher) EnrichSummary(
	ctx context.Context,
	summary []chrepo.CompanyRiskSummary,
) []AtRiskCompanySummary {
	if len(summary) == 0 {
		return []AtRiskCompanySummary{}
	}
	ids := make([]int32, len(summary))
	for i, s := range summary {
		ids[i] = s.CompanyID
	}
	compMap, _ := e.companies.GetByIDs(ctx, ids)
	out := make([]AtRiskCompanySummary, 0, len(summary))
	for _, s := range summary {
		row := AtRiskCompanySummary{CompanyRiskSummary: s}
		if c, ok := compMap[s.CompanyID]; ok {
			row.CompanyName = c.Name
		}
		out = append(out, row)
	}
	return out
}
