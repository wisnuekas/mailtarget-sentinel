package service

import (
	"context"
	"log/slog"

	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	"github.com/wisnuekas/mailtarget-sentinel/internal/mailtarget"
	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
)

// ResolveEmailRecipients returns the company owner's email from PostgreSQL
// (company.owner_id -> user.email). Falls back to ALERT_TO_EMAIL when lookup fails.
func ResolveEmailRecipients(
	ctx context.Context,
	companies *postgres.CompanyRepository,
	cfg *config.Config,
	companyID int32,
) []mailtarget.Address {
	owner, err := companies.GetOwnerByCompanyID(ctx, companyID)
	if err == nil && owner.Email != "" {
		name := owner.Name
		if name == "" {
			name = owner.Email
		}
		return []mailtarget.Address{{Email: owner.Email, Name: name}}
	}

	if err != nil {
		slog.Warn("alert recipient: company owner lookup failed, using fallback",
			"company_id", companyID,
			"error", err,
		)
	}

	return []mailtarget.Address{{Email: cfg.Alert.ToEmail, Name: cfg.Alert.ToName}}
}

// OpsTeamCC returns ALERT_TO_EMAIL / ALERT_TO_NAME as CC recipients for anomaly alerts.
func OpsTeamCC(cfg *config.Config) []mailtarget.Address {
	if cfg.Alert.ToEmail == "" {
		return nil
	}
	name := cfg.Alert.ToName
	if name == "" {
		name = "Ops Team"
	}
	return []mailtarget.Address{{Email: cfg.Alert.ToEmail, Name: name}}
}

// AnomalyEmailRecipients returns To (company owner) and CC (ops team), deduplicated.
func AnomalyEmailRecipients(
	ctx context.Context,
	companies *postgres.CompanyRepository,
	cfg *config.Config,
	companyID int32,
) (to, cc []mailtarget.Address) {
	to = ResolveEmailRecipients(ctx, companies, cfg, companyID)
	cc = OpsTeamCC(cfg)
	return to, dedupeCC(to, cc)
}

func dedupeCC(to, cc []mailtarget.Address) []mailtarget.Address {
	if len(cc) == 0 {
		return nil
	}
	toEmails := make(map[string]struct{}, len(to))
	for _, addr := range to {
		toEmails[addr.Email] = struct{}{}
	}
	out := make([]mailtarget.Address, 0, len(cc))
	for _, addr := range cc {
		if addr.Email == "" {
			continue
		}
		if _, dup := toEmails[addr.Email]; dup {
			continue
		}
		out = append(out, addr)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
