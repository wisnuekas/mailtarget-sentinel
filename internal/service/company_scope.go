package service

import (
	"context"

	"github.com/wisnuekas/mailtarget-sentinel/internal/config"
	redisstore "github.com/wisnuekas/mailtarget-sentinel/internal/redis"
)

const DefaultCompanyID int32 = 287

// ResolveCompanyScope: query param > Redis settings > COMPANY_ID env > default 287.
func ResolveCompanyScope(
	ctx context.Context,
	queryCompanyID *int32,
	cfg *config.Config,
	store *redisstore.Store,
) *int32 {
	if queryCompanyID != nil {
		return queryCompanyID
	}
	if store != nil {
		settings, err := store.GetSettings(ctx)
		if err == nil && settings.CompanyID != nil {
			if *settings.CompanyID > 0 {
				return settings.CompanyID
			}
			return nil
		}
	}
	if cfg.CompanyID > 0 {
		return &cfg.CompanyID
	}
	id := DefaultCompanyID
	return &id
}

// EffectiveCompanyID returns the configured company id for display (0 = all companies).
func EffectiveCompanyID(settings redisstore.Settings, cfg *config.Config) int32 {
	if settings.CompanyID != nil {
		return *settings.CompanyID
	}
	if cfg.CompanyID > 0 {
		return cfg.CompanyID
	}
	return DefaultCompanyID
}
