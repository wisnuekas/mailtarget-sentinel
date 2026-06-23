package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/wisnuekas/mailtarget-sentinel/internal/alert"
	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
)

// SendResumeConfirmationEmail emails the company owner a resume button after kill-switch suspend.
func SendResumeConfirmationEmail(
	ctx context.Context,
	cfg *config.Config,
	companies *postgres.CompanyRepository,
	transmission *mailtarget.TransmissionClient,
	store *redisstore.Store,
	subAccountID, companyID int32,
	alertID string,
) error {
	token, err := store.CreateResumeToken(ctx, redisstore.ResumeTokenPayload{
		SubAccountID: subAccountID,
		CompanyID:    companyID,
		AlertID:      alertID,
	})
	if err != nil {
		return fmt.Errorf("create resume token: %w", err)
	}

	email := alert.BuildResumeEmail(alert.ResumeEmailInput{
		SubAccountID: subAccountID,
		CompanyID:    companyID,
	}, cfg.PublicBaseURL, token)

	form := mailtarget.TransmissionForm{
		Subject:  email.Subject,
		From:     mailtarget.Address{Email: cfg.Alert.FromEmail, Name: cfg.Alert.FromName},
		To:       ResolveEmailRecipients(ctx, companies, cfg, companyID),
		BodyText: email.BodyText,
		BodyHTML: email.BodyHTML,
		Metadata: map[string]string{
			"sentinel_type":  "resume_confirmation",
			"sub_account_id": fmt.Sprintf("%d", subAccountID),
			"company_id":     fmt.Sprintf("%d", companyID),
			"alert_id":       alertID,
		},
		OptionsAttributes: &mailtarget.OptionsAttributes{
			ClickTracking: false,
			OpenTracking:  false,
			Transactional: true,
		},
	}

	result, err := transmission.Send(ctx, form)
	if err != nil {
		return fmt.Errorf("send resume email: %w", err)
	}

	slog.Info("resume confirmation email sent",
		"company_id", companyID,
		"sub_account_id", subAccountID,
		"transmission_id", result.TransmissionID,
	)
	return nil
}
