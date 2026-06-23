package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

type DomainRepository struct {
	pool *pgxpool.Pool
}

func NewDomainRepository(pool *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{pool: pool}
}

func (r *DomainRepository) FindByDomainNames(ctx context.Context, names []string, companyID *int32) (map[string]DomainRecord, error) {
	out := map[string]DomainRecord{}
	if len(names) == 0 {
		return out, nil
	}

	args := []interface{}{names}
	query := `
		SELECT id, domain, company_id, sub_account_id, is_sending, is_blocked
		FROM domain
		WHERE deleted_at IS NULL AND domain = ANY($1)`

	if companyID != nil {
		query += " AND company_id = $2"
		args = append(args, *companyID)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find domains by names: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var d DomainRecord
		var subID *int32
		if err := rows.Scan(&d.ID, &d.Domain, &d.CompanyID, &subID, &d.IsSending, &d.IsBlocked); err != nil {
			return nil, err
		}
		d.SubAccountID = subID
		out[d.Domain] = d
	}
	return out, rows.Err()
}
