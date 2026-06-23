package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/wisnuekas/mailtarget-sentinel/internal/alert"
	chrepo "github.com/wisnuekas/mailtarget-sentinel/internal/clickhouse"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
	"github.com/wisnuekas/mailtarget-sentinel/internal/service"
	"github.com/wisnuekas/mailtarget-sentinel/internal/sqlite"
)

const detectionWindow = 5 * time.Minute

type Detector struct {
	cfg          *config.Config
	events       *chrepo.EventRepository
	companies    *postgres.CompanyRepository
	store        *redisstore.Store
	alerts       *sqlite.AlertRepository
	transmission *mailtarget.TransmissionClient
	logger       *slog.Logger
}

func NewDetector(
	cfg *config.Config,
	events *chrepo.EventRepository,
	companies *postgres.CompanyRepository,
	store *redisstore.Store,
	alerts *sqlite.AlertRepository,
	transmission *mailtarget.TransmissionClient,
	logger *slog.Logger,
) *Detector {
	return &Detector{
		cfg:          cfg,
		events:       events,
		companies:    companies,
		store:        store,
		alerts:       alerts,
		transmission: transmission,
		logger:       logger,
	}
}

func (d *Detector) companyFilter(ctx context.Context) *int32 {
	return service.ResolveCompanyScope(ctx, nil, d.cfg, d.store)
}

func (d *Detector) companyScopeLabel(ctx context.Context) string {
	if id := d.companyFilter(ctx); id != nil {
		return fmt.Sprintf("%d", *id)
	}
	return "all"
}

func (d *Detector) Run(parent context.Context) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, d.cfg.WorkerTimeout)
	defer cancel()

	companyScope := d.companyScopeLabel(ctx)

	settings, err := d.store.GetSettings(ctx)
	if err != nil {
		d.logger.Error("worker: 5m check failed", "stage", "settings", "company_scope", companyScope, "error", err)
		return
	}

	d.logger.Info("worker: 5m check started",
		"window", detectionWindow.String(),
		"company_scope", companyScope,
		"min_volume", settings.MinVolume,
		"bounce_threshold_pct", settings.BounceRateThresholdPct,
		"spam_threshold_pct", settings.SpamRateThresholdPct,
	)

	anomalies, err := d.events.DetectAnomalies(
		ctx,
		d.companyFilter(ctx),
		detectionWindow,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		d.logger.Error("worker: 5m check failed", "stage", "sub_accounts", "company_scope", companyScope, "error", err)
		return
	}

	cooldown := time.Duration(settings.AlertCooldownMinutes) * time.Minute
	if len(anomalies) > 0 {
		for _, m := range anomalies {
			d.processAnomaly(ctx, m, cooldown)
		}
	}

	sendingIPs, err := d.events.DetectAtRiskSendingIPs(
		ctx,
		d.companyFilter(ctx),
		detectionWindow,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		d.logger.Error("worker: 5m check failed", "stage", "sending_ips", "company_scope", companyScope, "error", err)
		return
	}
	if len(sendingIPs) > 0 {
		for _, ip := range sendingIPs {
			d.processSendingIPAnomaly(ctx, ip, cooldown)
		}
	}

	d.logger.Info("worker: 5m check completed",
		"duration_ms", time.Since(started).Milliseconds(),
		"company_scope", companyScope,
		"sub_account_anomalies", len(anomalies),
		"at_risk_sending_ips", len(sendingIPs),
	)
}

func (d *Detector) processAnomaly(ctx context.Context, m chrepo.SubAccountMetrics, cooldown time.Duration) {
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

	// Always persist history for dashboard, even if email is deduplicated.
	if err := d.alerts.Create(ctx, sqlite.Alert{
		AlertID:       anomalyAlert.AlertID,
		CompanyID:     m.CompanyID,
		SubAccountID:  m.SubAccountID,
		Sent:          m.Sent,
		Bounced:       m.Bounced,
		SpamBounced:   m.SpamBounced,
		BounceRatePct: m.BounceRatePct,
		SpamRatePct:   m.SpamRatePct,
		Status:        sqlite.StatusDetected,
		DetectedAt:    anomalyAlert.DetectedAt,
	}); err != nil {
		d.logger.Error("worker: save alert history failed", "error", err)
	}

	acquired, err := d.store.TryAcquireAlertLock(ctx, m.SubAccountID, cooldown)
	if err != nil {
		d.logger.Error("worker: alert lock failed", "sub_account_id", m.SubAccountID, "error", err)
		return
	}
	if !acquired {
		d.logger.Debug("worker: alert email deduplicated", "sub_account_id", m.SubAccountID)
		return
	}

	token, err := d.store.CreateKillToken(ctx, redisstore.KillTokenPayload{
		SubAccountID: m.SubAccountID,
		Alert:        anomalyAlert,
	})
	if err != nil {
		d.logger.Error("worker: create kill token failed", "sub_account_id", m.SubAccountID, "error", err)
		return
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
		},
		OptionsAttributes: &mailtarget.OptionsAttributes{
			ClickTracking: false,
			OpenTracking:  false,
			Transactional: true,
		},
	}

	result, err := d.transmission.Send(ctx, form)
	if err != nil {
		d.logger.Error("worker: send alert email failed",
			"sub_account_id", m.SubAccountID,
			"company_id", m.CompanyID,
			"error", err,
		)
		return
	}

	txID := result.TransmissionID
	if err := d.alerts.UpdateStatus(ctx, anomalyAlert.AlertID, sqlite.StatusAlertSent, &txID); err != nil {
		d.logger.Error("worker: update alert status failed", "error", err)
	}

	d.logger.Info("worker: alert email sent",
		"company_id", m.CompanyID,
		"sub_account_id", m.SubAccountID,
		"transmission_id", result.TransmissionID,
		"bounce_rate", m.BounceRatePct,
		"spam_rate", m.SpamRatePct,
	)
}

func (d *Detector) processSendingIPAnomaly(ctx context.Context, m chrepo.SendingIPMetrics, cooldown time.Duration) {
	acquired, err := d.store.TryAcquireSendingIPAlertLock(ctx, m.SendingIP, cooldown)
	if err != nil {
		d.logger.Error("worker: sending IP alert lock failed",
			"sending_ip", m.SendingIP,
			"error", err,
		)
		return
	}
	if !acquired {
		d.logger.Debug("worker: sending IP alert deduplicated", "sending_ip", m.SendingIP)
		return
	}

	email := alert.BuildSendingIPEmail(alert.SendingIPEmailInput{
		SendingIP:         m.SendingIP,
		AffectedCompanies: m.AffectedCompanies,
		Sent:              m.Sent,
		Bounced:           m.Bounced,
		SpamBounced:       m.SpamBounced,
		BounceRate:        m.BounceRatePct,
		SpamRate:          m.SpamRatePct,
	})

	form := mailtarget.TransmissionForm{
		Subject:  email.Subject,
		From:     mailtarget.Address{Email: d.cfg.Alert.FromEmail, Name: d.cfg.Alert.FromName},
		To:       service.OpsTeamCC(d.cfg),
		BodyText: email.BodyText,
		BodyHTML: email.BodyHTML,
		Metadata: map[string]string{
			"sentinel_type":        "sending_ip_review",
			"sending_ip":           m.SendingIP,
			"affected_companies":   fmt.Sprintf("%d", m.AffectedCompanies),
		},
		OptionsAttributes: &mailtarget.OptionsAttributes{
			ClickTracking: false,
			OpenTracking:  false,
			Transactional: true,
		},
	}

	if len(form.To) == 0 {
		d.logger.Warn("worker: sending IP alert skipped, no OpsTeam recipient configured",
			"sending_ip", m.SendingIP,
		)
		return
	}

	result, err := d.transmission.Send(ctx, form)
	if err != nil {
		d.logger.Error("worker: send sending IP alert failed",
			"sending_ip", m.SendingIP,
			"error", err,
		)
		return
	}

	d.logger.Info("worker: sending IP alert email sent",
		"sending_ip", m.SendingIP,
		"affected_companies", m.AffectedCompanies,
		"transmission_id", result.TransmissionID,
		"bounce_rate", m.BounceRatePct,
		"spam_rate", m.SpamRatePct,
	)
}
