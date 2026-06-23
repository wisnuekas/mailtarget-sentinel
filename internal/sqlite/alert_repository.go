package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	StatusDetected  = "detected"
	StatusAlertSent = "alert_sent"
	StatusSuspended = "suspended"
	StatusResolved  = "resolved"
)

type Alert struct {
	ID             int64      `json:"id"`
	AlertID        string     `json:"alert_id"`
	CompanyID      int32      `json:"company_id"`
	SubAccountID   int32      `json:"sub_account_id"`
	Sent           uint64     `json:"sent"`
	Bounced        uint64     `json:"bounced"`
	SpamBounced    uint64     `json:"spam_bounced"`
	BounceRatePct  float64    `json:"bounce_rate_pct"`
	SpamRatePct    float64    `json:"spam_rate_pct"`
	Status         string     `json:"status"`
	TransmissionID *string    `json:"transmission_id,omitempty"`
	DetectedAt     time.Time  `json:"detected_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
}

type AlertFilter struct {
	CompanyID    *int32
	SubAccountID *int32
	Status       string
	From         *time.Time
	To           *time.Time
	Page         int
	Limit        int
}

type AlertRepository struct {
	db *sql.DB
}

func NewAlertRepository(db *sql.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Create(ctx context.Context, a Alert) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO alerts (
    alert_id, company_id, sub_account_id, sent, bounced, spam_bounced,
    bounce_rate_pct, spam_rate_pct, status, transmission_id, detected_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.AlertID, a.CompanyID, a.SubAccountID, a.Sent, a.Bounced, a.SpamBounced,
		a.BounceRatePct, a.SpamRatePct, a.Status, a.TransmissionID, a.DetectedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	return nil
}

func (r *AlertRepository) GetByAlertID(ctx context.Context, alertID string) (*Alert, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, alert_id, company_id, sub_account_id, sent, bounced, spam_bounced,
       bounce_rate_pct, spam_rate_pct, status, transmission_id, detected_at, resolved_at
FROM alerts WHERE alert_id = ?`, alertID)

	return scanAlert(row)
}

func (r *AlertRepository) List(ctx context.Context, f AlertFilter) ([]Alert, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}

	where, args := buildAlertWhere(f)

	var total int
	countQuery := "SELECT COUNT(*) FROM alerts" + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	listArgs := append(args, f.Limit, offset)
	query := `
SELECT id, alert_id, company_id, sub_account_id, sent, bounced, spam_bounced,
       bounce_rate_pct, spam_rate_pct, status, transmission_id, detected_at, resolved_at
FROM alerts` + where + `
ORDER BY detected_at DESC
LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		alerts = append(alerts, *a)
	}

	return alerts, total, rows.Err()
}

func (r *AlertRepository) UpdateStatus(ctx context.Context, alertID, status string, transmissionID *string) error {
	if transmissionID != nil {
		_, err := r.db.ExecContext(ctx, `
UPDATE alerts SET status = ?, transmission_id = ? WHERE alert_id = ?`,
			status, *transmissionID, alertID,
		)
		return err
	}

	var resolvedAt interface{}
	if status == StatusSuspended || status == StatusResolved {
		resolvedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, `
UPDATE alerts SET status = ?, resolved_at = COALESCE(?, resolved_at) WHERE alert_id = ?`,
		status, resolvedAt, alertID,
	)
	if err != nil {
		return fmt.Errorf("update alert status: %w", err)
	}
	return nil
}

func (r *AlertRepository) UpdateStatusBySubAccount(ctx context.Context, subAccountID int32, status string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
UPDATE alerts SET status = ?, resolved_at = ?
WHERE sub_account_id = ? AND status IN ('detected', 'alert_sent')
  AND id = (
    SELECT id FROM alerts
    WHERE sub_account_id = ? AND status IN ('detected', 'alert_sent')
    ORDER BY detected_at DESC LIMIT 1
  )`, status, now, subAccountID, subAccountID)
	if err != nil {
		return fmt.Errorf("update alert by sub_account: %w", err)
	}
	return nil
}

func (r *AlertRepository) Recent(ctx context.Context, limit int) ([]Alert, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, alert_id, company_id, sub_account_id, sent, bounced, spam_bounced,
       bounce_rate_pct, spam_rate_pct, status, transmission_id, detected_at, resolved_at
FROM alerts ORDER BY detected_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, *a)
	}
	return alerts, rows.Err()
}

func (r *AlertRepository) Stats(ctx context.Context) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT status, COUNT(*) FROM alerts GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, rows.Err()
}

func buildAlertWhere(f AlertFilter) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if f.CompanyID != nil {
		clauses = append(clauses, "company_id = ?")
		args = append(args, *f.CompanyID)
	}
	if f.SubAccountID != nil {
		clauses = append(clauses, "sub_account_id = ?")
		args = append(args, *f.SubAccountID)
	}
	if f.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, f.Status)
	}
	if f.From != nil {
		clauses = append(clauses, "detected_at >= ?")
		args = append(args, f.From.UTC())
	}
	if f.To != nil {
		clauses = append(clauses, "detected_at <= ?")
		args = append(args, f.To.UTC())
	}

	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanAlert(row scannable) (*Alert, error) {
	var a Alert
	var transmissionID sql.NullString
	var resolvedAt sql.NullTime

	err := row.Scan(
		&a.ID, &a.AlertID, &a.CompanyID, &a.SubAccountID,
		&a.Sent, &a.Bounced, &a.SpamBounced,
		&a.BounceRatePct, &a.SpamRatePct, &a.Status,
		&transmissionID, &a.DetectedAt, &resolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan alert: %w", err)
	}

	if transmissionID.Valid {
		a.TransmissionID = &transmissionID.String
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		a.ResolvedAt = &t
	}

	return &a, nil
}
