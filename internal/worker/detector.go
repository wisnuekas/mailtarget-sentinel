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

func (d *Detector) Run(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, d.cfg.WorkerTimeout)
	defer cancel()

	settings, err := d.store.GetSettings(ctx)
	if err != nil {
		d.logger.Error("worker: load settings failed", "error", err)
		return
	}

	anomalies, err := d.events.DetectAnomalies(
		ctx,
		d.companyFilter(ctx),
		detectionWindow,
		settings.MinVolume,
		settings.BounceRateThresholdPct,
		settings.SpamRateThresholdPct,
	)
	if err != nil {
		d.logger.Error("worker: detect anomalies failed", "error", err)
		return
	}

	cooldown := time.Duration(settings.AlertCooldownMinutes) * time.Minute
	if len(anomalies) > 0 {
		d.logger.Info("worker: anomalies detected", "count", len(anomalies))
		for _, m := range anomalies {
			d.processAnomaly(ctx, m, cooldown)
		}
	} else {
		companyScope := "all"
		if id := d.companyFilter(ctx); id != nil {
			companyScope = fmt.Sprintf("%d", *id)
		}
		d.logger.Info("worker: no sub-account anomalies detected",
			"company_scope", companyScope,
			"window", detectionWindow.String(),
			"min_volume", settings.MinVolume,
			"bounce_threshold_pct", settings.BounceRateThresholdPct,
		)
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
		d.logger.Error("worker: detect at-risk sending IPs failed", "error", err)
		return
	}
	if len(sendingIPs) > 0 {
		d.logger.Info("worker: at-risk sending IPs detected", "count", len(sendingIPs))
	}
	for _, ip := range sendingIPs {
		d.processSendingIPAnomaly(ctx, ip, cooldown)
	}
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
		d.logger.Info("worker: alert email deduplicated", "sub_account_id", m.SubAccountID)
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
	acquired, err := d.store.TryAcquireSendingIPAlertLock(ctx, m.CompanyID, m.SendingIP, cooldown)
	if err != nil {
		d.logger.Error("worker: sending IP alert lock failed",
			"company_id", m.CompanyID,
			"sending_ip", m.SendingIP,
			"error", err,
		)
		return
	}
	if !acquired {
		d.logger.Info("worker: sending IP alert deduplicated",
			"company_id", m.CompanyID,
			"sending_ip", m.SendingIP,
		)
		return
	}

	companyName := ""
	if co, err := d.companies.GetByID(ctx, m.CompanyID); err == nil {
		companyName = co.Name
	}

	email := alert.BuildSendingIPEmail(alert.SendingIPEmailInput{
		CompanyID:   m.CompanyID,
		CompanyName: companyName,
		SendingIP:   m.SendingIP,
		Sent:        m.Sent,
		Bounced:     m.Bounced,
		SpamBounced: m.SpamBounced,
		BounceRate:  m.BounceRatePct,
		SpamRate:    m.SpamRatePct,
	})

	form := mailtarget.TransmissionForm{
		Subject:  email.Subject,
		From:     mailtarget.Address{Email: d.cfg.Alert.FromEmail, Name: d.cfg.Alert.FromName},
		To:       service.OpsTeamCC(d.cfg),
		BodyText: email.BodyText,
		BodyHTML: email.BodyHTML,
		Metadata: map[string]string{
			"sentinel_type": "sending_ip_review",
			"company_id":    fmt.Sprintf("%d", m.CompanyID),
			"sending_ip":    m.SendingIP,
		},
		OptionsAttributes: &mailtarget.OptionsAttributes{
			ClickTracking: false,
			OpenTracking:  false,
			Transactional: true,
		},
	}

	if len(form.To) == 0 {
		d.logger.Warn("worker: sending IP alert skipped, no OpsTeam recipient configured",
			"company_id", m.CompanyID,
			"sending_ip", m.SendingIP,
		)
		return
	}

	result, err := d.transmission.Send(ctx, form)
	if err != nil {
		d.logger.Error("worker: send sending IP alert failed",
			"company_id", m.CompanyID,
			"sending_ip", m.SendingIP,
			"error", err,
		)
		return
	}

	d.logger.Info("worker: sending IP alert email sent",
		"company_id", m.CompanyID,
		"sending_ip", m.SendingIP,
		"transmission_id", result.TransmissionID,
		"bounce_rate", m.BounceRatePct,
		"spam_rate", m.SpamRatePct,
	)
}
