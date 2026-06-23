package handler

import (
	"context"
	"fmt"

	"github.com/wisnuekas/mailtarget-sentinel/internal/postgres"
)

// applySubAccountStatus updates sub_account.status in PostgreSQL (dashboard DB).
func applySubAccountStatus(
	ctx context.Context,
	repo *postgres.SubAccountRepository,
	subAccountID int32,
	action string,
) (string, error) {
	var status string
	switch action {
	case "suspend":
		status = postgres.StatusSuspended
	case "resume":
		status = postgres.StatusActive
	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
	if err := repo.UpdateStatus(ctx, subAccountID, status); err != nil {
		return "", err
	}
	return status, nil
}
