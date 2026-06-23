package alert

import "time"

type AnomalyAlert struct {
	AlertID      string    `json:"alert_id"`
	CompanyID    int32     `json:"company_id"`
	SubAccountID int32     `json:"sub_account_id"`
	Sent         uint64    `json:"sent"`
	Bounced      uint64    `json:"bounced"`
	SpamBounced  uint64    `json:"spam_bounced"`
	BounceRate   float64   `json:"bounce_rate_pct"`
	SpamRate     float64   `json:"spam_rate_pct"`
	DetectedAt   time.Time `json:"detected_at"`
}
