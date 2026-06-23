package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type SubAccountMetrics struct {
	CompanyID       int32   `json:"company_id"`
	SubAccountID    int32   `json:"sub_account_id"`
	Sent            uint64  `json:"sent"`
	Delivered       uint64  `json:"delivered"`
	Bounced         uint64  `json:"bounced"`
	SpamBounced     uint64  `json:"spam_bounced"`
	BounceRatePct   float64 `json:"bounce_rate_pct"`
	DeliveryRatePct float64 `json:"delivery_rate_pct"`
	SpamRatePct     float64 `json:"spam_rate_pct"`
}

type DomainMetrics struct {
	CompanyID       int32   `json:"company_id"`
	SendingDomain   string  `json:"sending_domain"`
	Sent            uint64  `json:"sent"`
	Delivered       uint64  `json:"delivered"`
	Bounced         uint64  `json:"bounced"`
	SpamBounced     uint64  `json:"spam_bounced"`
	BounceRatePct   float64 `json:"bounce_rate_pct"`
	DeliveryRatePct float64 `json:"delivery_rate_pct"`
	SpamRatePct     float64 `json:"spam_rate_pct"`
}

type SendingIPMetrics struct {
	CompanyID       int32   `json:"company_id"`
	SendingIP       string  `json:"sending_ip"`
	Sent            uint64  `json:"sent"`
	Delivered       uint64  `json:"delivered"`
	Bounced         uint64  `json:"bounced"`
	SpamBounced     uint64  `json:"spam_bounced"`
	BounceRatePct   float64 `json:"bounce_rate_pct"`
	DeliveryRatePct float64 `json:"delivery_rate_pct"`
	SpamRatePct     float64 `json:"spam_rate_pct"`
}

type CompanyRiskSummary struct {
	CompanyID         int32   `json:"company_id"`
	AffectedAccounts  uint64  `json:"affected_accounts"`
	TotalSent         uint64  `json:"total_sent"`
	MaxBounceRatePct  float64 `json:"max_bounce_rate_pct"`
	MaxSpamRatePct    float64 `json:"max_spam_rate_pct"`
	WorstSubAccountID int32   `json:"worst_sub_account_id"`
}

type EventRepository struct {
	conn clickhouse.Conn
}

func NewEventRepository(conn clickhouse.Conn) *EventRepository {
	return &EventRepository{conn: conn}
}

func windowSeconds(window time.Duration) int {
	secs := int(window.Seconds())
	if secs <= 0 {
		return 300
	}
	return secs
}

func (r *EventRepository) GetMetrics(ctx context.Context, companyID *int32, window time.Duration, subAccountID *int32) ([]SubAccountMetrics, error) {
	secs := windowSeconds(window)

	query := fmt.Sprintf(`
SELECT
    company_id,
    sub_account_id,
    countIf(type = 'injection') AS sent,
    countIf(type = 'delivery') AS delivered,
    countIf(type = 'bounce') AS bounced,
    countIf(type = 'bounce' AND bounce_classification_code IN (50, 51, 52, 53, 54)) AS spam_bounced
FROM default.event
WHERE injection_time >= now() - INTERVAL %d SECOND
  AND injection_time < now()
  AND sub_account_id > 0
`, secs)

	args := []interface{}{}

	if companyID != nil {
		query += " AND company_id = ?"
		args = append(args, *companyID)
	}
	if subAccountID != nil {
		query += " AND sub_account_id = ?"
		args = append(args, *subAccountID)
	}

	query += `
GROUP BY company_id, sub_account_id
ORDER BY sent DESC
`

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return emptyMetrics(), fmt.Errorf("get metrics query: %w", err)
	}
	defer rows.Close()

	results, err := scanMetricsRows(rows)
	if err != nil {
		return emptyMetrics(), err
	}
	return results, nil
}

func (r *EventRepository) DetectAnomalies(
	ctx context.Context,
	companyID *int32,
	window time.Duration,
	minVolume uint64,
	bounceThresholdPct float64,
	spamThresholdPct float64,
) ([]SubAccountMetrics, error) {
	secs := windowSeconds(window)

	companyFilter := ""
	args := []interface{}{}
	if companyID != nil {
		companyFilter = " AND company_id = ?"
		args = append(args, *companyID)
	}
	args = append(args, minVolume, bounceThresholdPct, spamThresholdPct)

	query := fmt.Sprintf(`
SELECT
    company_id,
    sub_account_id,
    sent,
    delivered,
    bounced,
    spam_bounced,
    if(sent > 0, bounced / sent * 100, 0) AS bounce_rate_pct,
    if(sent > 0, delivered / sent * 100, 0) AS delivery_rate_pct,
    if(sent > 0, spam_bounced / sent * 100, 0) AS spam_rate_pct
FROM (
    SELECT
        company_id,
        sub_account_id,
        countIf(type = 'injection') AS sent,
        countIf(type = 'delivery') AS delivered,
        countIf(type = 'bounce') AS bounced,
        countIf(type = 'bounce' AND bounce_classification_code IN (50, 51, 52, 53, 54)) AS spam_bounced
    FROM default.event
    WHERE injection_time >= now() - INTERVAL %d SECOND
      AND injection_time < now()
      AND sub_account_id > 0%s
    GROUP BY company_id, sub_account_id
    HAVING sent >= ?
)
WHERE bounce_rate_pct > ? OR spam_rate_pct > ?
ORDER BY bounce_rate_pct DESC
`, secs, companyFilter)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return emptyMetrics(), fmt.Errorf("detect anomalies query: %w", err)
	}
	defer rows.Close()

	var results []SubAccountMetrics
	for rows.Next() {
		var m SubAccountMetrics
		if err := rows.Scan(
			&m.CompanyID,
			&m.SubAccountID,
			&m.Sent,
			&m.Delivered,
			&m.Bounced,
			&m.SpamBounced,
			&m.BounceRatePct,
			&m.DeliveryRatePct,
			&m.SpamRatePct,
		); err != nil {
			return emptyMetrics(), fmt.Errorf("scan anomaly row: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return emptyMetrics(), err
	}
	if results == nil {
		return emptyMetrics(), nil
	}
	return results, nil
}

func (r *EventRepository) DetectAtRiskDomains(
	ctx context.Context,
	companyID *int32,
	window time.Duration,
	minVolume uint64,
	bounceThresholdPct float64,
	spamThresholdPct float64,
) ([]DomainMetrics, error) {
	secs := windowSeconds(window)

	companyFilter := ""
	args := []interface{}{}
	if companyID != nil {
		companyFilter = " AND company_id = ?"
		args = append(args, *companyID)
	}
	args = append(args, minVolume, bounceThresholdPct, spamThresholdPct)

	query := fmt.Sprintf(`
SELECT
    company_id,
    sending_domain,
    sent,
    delivered,
    bounced,
    spam_bounced,
    if(sent > 0, bounced / sent * 100, 0) AS bounce_rate_pct,
    if(sent > 0, delivered / sent * 100, 0) AS delivery_rate_pct,
    if(sent > 0, spam_bounced / sent * 100, 0) AS spam_rate_pct
FROM (
    SELECT
        company_id,
        sending_domain,
        countIf(type = 'injection') AS sent,
        countIf(type = 'delivery') AS delivered,
        countIf(type = 'bounce') AS bounced,
        countIf(type = 'bounce' AND bounce_classification_code IN (50, 51, 52, 53, 54)) AS spam_bounced
    FROM default.event
    WHERE injection_time >= now() - INTERVAL %d SECOND
      AND injection_time < now()
      AND sending_domain != ''%s
    GROUP BY company_id, sending_domain
    HAVING sent >= ?
)
WHERE bounce_rate_pct > ? OR spam_rate_pct > ?
ORDER BY bounce_rate_pct DESC
`, secs, companyFilter)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return emptyDomains(), fmt.Errorf("detect at-risk domains query: %w", err)
	}
	defer rows.Close()

	var results []DomainMetrics
	for rows.Next() {
		var m DomainMetrics
		if err := rows.Scan(
			&m.CompanyID,
			&m.SendingDomain,
			&m.Sent,
			&m.Delivered,
			&m.Bounced,
			&m.SpamBounced,
			&m.BounceRatePct,
			&m.DeliveryRatePct,
			&m.SpamRatePct,
		); err != nil {
			return emptyDomains(), fmt.Errorf("scan at-risk domain row: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return emptyDomains(), err
	}
	if results == nil {
		return emptyDomains(), nil
	}
	return results, nil
}

func (r *EventRepository) DetectAtRiskSendingIPs(
	ctx context.Context,
	companyID *int32,
	window time.Duration,
	minVolume uint64,
	bounceThresholdPct float64,
	spamThresholdPct float64,
) ([]SendingIPMetrics, error) {
	secs := windowSeconds(window)

	companyFilter := ""
	args := []interface{}{}
	if companyID != nil {
		companyFilter = " AND company_id = ?"
		args = append(args, *companyID)
	}
	args = append(args, minVolume, bounceThresholdPct, spamThresholdPct)

	query := fmt.Sprintf(`
SELECT
    company_id,
    sending_ip,
    sent,
    delivered,
    bounced,
    spam_bounced,
    if(sent > 0, bounced / sent * 100, 0) AS bounce_rate_pct,
    if(sent > 0, delivered / sent * 100, 0) AS delivery_rate_pct,
    if(sent > 0, spam_bounced / sent * 100, 0) AS spam_rate_pct
FROM (
    SELECT
        company_id,
        sending_ip,
        countIf(type = 'injection') AS sent,
        countIf(type = 'delivery') AS delivered,
        countIf(type = 'bounce') AS bounced,
        countIf(type = 'bounce' AND bounce_classification_code IN (50, 51, 52, 53, 54)) AS spam_bounced
    FROM default.event
    WHERE injection_time >= now() - INTERVAL %d SECOND
      AND injection_time < now()
      AND sending_ip IS NOT NULL
      AND sending_ip != ''%s
    GROUP BY company_id, sending_ip
    HAVING sent >= ?
)
WHERE bounce_rate_pct > ? OR spam_rate_pct > ?
ORDER BY bounce_rate_pct DESC
`, secs, companyFilter)

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return emptySendingIPs(), fmt.Errorf("detect at-risk sending IPs query: %w", err)
	}
	defer rows.Close()

	var results []SendingIPMetrics
	for rows.Next() {
		var m SendingIPMetrics
		if err := rows.Scan(
			&m.CompanyID,
			&m.SendingIP,
			&m.Sent,
			&m.Delivered,
			&m.Bounced,
			&m.SpamBounced,
			&m.BounceRatePct,
			&m.DeliveryRatePct,
			&m.SpamRatePct,
		); err != nil {
			return emptySendingIPs(), fmt.Errorf("scan at-risk sending IP row: %w", err)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return emptySendingIPs(), err
	}
	if results == nil {
		return emptySendingIPs(), nil
	}
	return results, nil
}

func (r *EventRepository) GetAtRiskSummary(
	ctx context.Context,
	window time.Duration,
	minVolume uint64,
	bounceThresholdPct float64,
	spamThresholdPct float64,
) ([]CompanyRiskSummary, error) {
	return r.GetAtRiskSummaryForCompany(ctx, nil, window, minVolume, bounceThresholdPct, spamThresholdPct)
}

func (r *EventRepository) GetAtRiskSummaryForCompany(
	ctx context.Context,
	companyID *int32,
	window time.Duration,
	minVolume uint64,
	bounceThresholdPct float64,
	spamThresholdPct float64,
) ([]CompanyRiskSummary, error) {
	anomalies, err := r.DetectAnomalies(ctx, companyID, window, minVolume, bounceThresholdPct, spamThresholdPct)
	if err != nil {
		return emptySummary(), err
	}

	byCompany := map[int32]*CompanyRiskSummary{}
	for _, m := range anomalies {
		s, ok := byCompany[m.CompanyID]
		if !ok {
			byCompany[m.CompanyID] = &CompanyRiskSummary{
				CompanyID:         m.CompanyID,
				AffectedAccounts:  1,
				TotalSent:         m.Sent,
				MaxBounceRatePct:  m.BounceRatePct,
				MaxSpamRatePct:    m.SpamRatePct,
				WorstSubAccountID: m.SubAccountID,
			}
			continue
		}
		s.AffectedAccounts++
		s.TotalSent += m.Sent
		if m.BounceRatePct > s.MaxBounceRatePct {
			s.MaxBounceRatePct = m.BounceRatePct
			s.WorstSubAccountID = m.SubAccountID
		}
		if m.SpamRatePct > s.MaxSpamRatePct {
			s.MaxSpamRatePct = m.SpamRatePct
		}
	}

	results := make([]CompanyRiskSummary, 0, len(byCompany))
	for _, s := range byCompany {
		s.MaxBounceRatePct = round2(s.MaxBounceRatePct)
		s.MaxSpamRatePct = round2(s.MaxSpamRatePct)
		results = append(results, *s)
	}

	return results, nil
}

func emptyMetrics() []SubAccountMetrics {
	return []SubAccountMetrics{}
}

func emptyDomains() []DomainMetrics {
	return []DomainMetrics{}
}

func emptySendingIPs() []SendingIPMetrics {
	return []SendingIPMetrics{}
}

func emptySummary() []CompanyRiskSummary {
	return []CompanyRiskSummary{}
}

func scanMetricsRows(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]SubAccountMetrics, error) {
	results := []SubAccountMetrics{}
	for rows.Next() {
		var m SubAccountMetrics
		if err := rows.Scan(&m.CompanyID, &m.SubAccountID, &m.Sent, &m.Delivered, &m.Bounced, &m.SpamBounced); err != nil {
			return emptyMetrics(), fmt.Errorf("scan metrics row: %w", err)
		}
		if m.Sent > 0 {
			m.BounceRatePct = round2(float64(m.Bounced) / float64(m.Sent) * 100)
			m.DeliveryRatePct = round2(float64(m.Delivered) / float64(m.Sent) * 100)
			m.SpamRatePct = round2(float64(m.SpamBounced) / float64(m.Sent) * 100)
		}
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return emptyMetrics(), err
	}
	return results, nil
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
