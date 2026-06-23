package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wisnuekas/mailtarget-sentinel/internal/alert"
	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
)

type KillSwitchEmailResult struct {
	AlertID        string `json:"alert_id"`
	SubAccountID   int32  `json:"sub_account_id"`
	TransmissionID string `json:"transmission_id"`
	Subject        string `json:"subject"`
}

// ResendKillSwitchEmail sends a fresh anomaly alert email with a new kill-switch token.
// Unlike the cron worker, this bypasses Redis dedup so operators can manually resend.
func (d *Detector) ResendKillSwitchEmail(ctx context.Context, subAccountID int32, window time.Duration) (*KillSwitchEmailResult, error) {
	subID := subAccountID
	metrics, err := d.events.GetMetrics(ctx, d.companyFilter(ctx), window, &subID)
	if err != nil {
		return nil, fmt.Errorf("load metrics: %w", err)
	}
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no metrics for sub_account %d in window %s", subAccountID, window)
	}

	return d.deliverKillSwitchEmail(ctx, metrics[0])
}

// ResendKillSwitchEmailByAlertID resends using metrics stored on an existing alert row.
func (d *Detector) ResendKillSwitchEmailByAlertID(ctx context.Context, alertID string) (*KillSwitchEmailResult, error) {
	existing, err := d.alerts.GetByAlertID(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("alert not found")
	}

	m := chrepo.SubAccountMetrics{
		CompanyID:     existing.CompanyID,
		SubAccountID:  existing.SubAccountID,
		Sent:          existing.Sent,
		Bounced:       existing.Bounced,
		SpamBounced:   existing.SpamBounced,
		BounceRatePct: existing.BounceRatePct,
		SpamRatePct:   existing.SpamRatePct,
	}

	return d.deliverKillSwitchEmail(ctx, m)
}

func (d *Detector) deliverKillSwitchEmail(ctx context.Context, m chrepo.SubAccountMetrics) (*KillSwitchEmailResult, error) {
	anomalyAlert := alert.AnomalyAlert{
		AlertID:      uuid.New().String(),
		CompanyID:    m.CompanyID,
		SubAccountID: m.SubAccountID,
		Sent:         m.Sent,
		Bounced:      m.Bounced,
		SpamBounced:  m.SpamBounced,
		BounceRate:   m.BounceRatePct,
		SpamRate:     m.SpamRatePct,
		DetectedAt:   time.Now().UTC(),
	}

	token, err := d.store.CreateKillToken(ctx, redisstore.KillTokenPayload{
		SubAccountID: m.SubAccountID,
		Alert:        anomalyAlert,
	})
	if err != nil {
		return nil, fmt.Errorf("create kill token: %w", err)
	}

	email := alert.BuildAlertEmail(anomalyAlert, d.cfg.PublicBaseURL, token)

	to, cc := service.AnomalyEmailRecipients(ctx, d.companies, d.cfg, m.CompanyID)
	form := mailtarget.TransmissionForm{
		Subject:  email.Subject,
		From:     mailtarget.Address{Email: d.cfg.Alert.FromEmail, Name: d.cfg.Alert.FromName},
		To:       to,
		CC:       cc,
		BodyText: email.BodyText,
		BodyHTML: email.BodyHTML,
		Metadata: map[string]string{
			"sentinel_alert_id": anomalyAlert.AlertID,
			"sub_account_id":    fmt.Sprintf("%d", m.SubAccountID),
			"company_id":        fmt.Sprintf("%d", m.CompanyID),
			"sentinel_resend":   "true",
		},
		OptionsAttributes: &mailtarget.OptionsAttributes{
			ClickTracking: false,
			OpenTracking:  false,
			Transactional: true,
		},
	}

	result, err := d.transmission.Send(ctx, form)
	if err != nil {
		return nil, fmt.Errorf("send alert email: %w", err)
	}

	txID := result.TransmissionID
	if err := d.alerts.Create(ctx, sqlite.Alert{
		AlertID:        anomalyAlert.AlertID,
		CompanyID:      m.CompanyID,
		SubAccountID:   m.SubAccountID,
		Sent:           m.Sent,
		Bounced:        m.Bounced,
		SpamBounced:    m.SpamBounced,
		BounceRatePct:  m.BounceRatePct,
		SpamRatePct:    m.SpamRatePct,
		Status:         sqlite.StatusAlertSent,
		TransmissionID: &txID,
		DetectedAt:     anomalyAlert.DetectedAt,
	}); err != nil {
		d.logger.Error("resend: save alert history failed", "error", err)
	}

	d.logger.Info("kill-switch alert email resent",
		"company_id", m.CompanyID,
		"sub_account_id", m.SubAccountID,
		"transmission_id", result.TransmissionID,
		"alert_id", anomalyAlert.AlertID,
	)

	return &KillSwitchEmailResult{
		AlertID:        anomalyAlert.AlertID,
		SubAccountID:   m.SubAccountID,
		TransmissionID: result.TransmissionID,
		Subject:        email.Subject,
	}, nil
}
