package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type SubAccountRepository struct {
	pool *pgxpool.Pool
}

func NewSubAccountRepository(pool *pgxpool.Pool) *SubAccountRepository {
	return &SubAccountRepository{pool: pool}
}

type SubAccountListFilter struct {
	CompanyID *int32
	Search    string
	Status    string
	Page      int
	Size      int
}

func (r *SubAccountRepository) List(ctx context.Context, f SubAccountListFilter) ([]SubAccount, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size <= 0 {
		f.Size = 20
	}
	offset := (f.Page - 1) * f.Size

	where := []string{
		"sa.deleted_at IS NULL",
		"sa.name <> 'sandbox'",
	}
	args := []interface{}{}
	argN := 1

	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("sa.company_id = $%d", argN))
		args = append(args, *f.CompanyID)
		argN++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("sa.name ILIKE $%d", argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}
	if f.Status != "" {
		where = append(where, fmt.Sprintf("sa.status = $%d", argN))
		args = append(args, f.Status)
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int
	countQ := fmt.Sprintf(`
		SELECT COUNT(DISTINCT sa.id)
		FROM sub_account sa
		WHERE %s`, whereSQL)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count sub_accounts: %w", err)
	}

	listArgs := append(args, f.Size, offset)
	query := fmt.Sprintf(`
		SELECT sa.id, sa.company_id, sa.name, sa.status,
		       COALESCE(ip.name, ''),
		       FLOOR(EXTRACT(EPOCH FROM COALESCE(sa.created_at, CURRENT_TIMESTAMP))::bigint * 1000),
		       COALESCE(c.name, '')
		FROM sub_account sa
		LEFT JOIN ip_pool ip ON ip.id = sa.ip_pool_id AND ip.deleted_at IS NULL
		LEFT JOIN company c ON c.id = sa.company_id AND c.deleted_at IS NULL
		WHERE %s
		ORDER BY sa.created_at DESC
		LIMIT $%d OFFSET $%d`, whereSQL, argN, argN+1)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list sub_accounts: %w", err)
	}
	defer rows.Close()

	var list []SubAccount
	for rows.Next() {
		var sa SubAccount
		if err := rows.Scan(&sa.ID, &sa.CompanyID, &sa.Name, &sa.Status, &sa.IPPoolName, &sa.CreatedAt, &sa.CompanyName); err != nil {
			return nil, 0, err
		}
		list = append(list, sa)
	}
	return list, total, rows.Err()
}

func (r *SubAccountRepository) GetByID(ctx context.Context, id int32) (*SubAccountDetail, error) {
	query := `
		SELECT sa.id, sa.company_id, sa.name, sa.status, sa.ip_pool_id,
		       COALESCE(ip.name, ''),
		       FLOOR(EXTRACT(EPOCH FROM COALESCE(sa.created_at, CURRENT_TIMESTAMP))::bigint * 1000),
		       COALESCE(c.name, '')
		FROM sub_account sa
		LEFT JOIN ip_pool ip ON ip.id = sa.ip_pool_id AND ip.deleted_at IS NULL
		LEFT JOIN company c ON c.id = sa.company_id AND c.deleted_at IS NULL
		WHERE sa.deleted_at IS NULL AND sa.id = $1`

	var detail SubAccountDetail
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&detail.ID, &detail.CompanyID, &detail.Name, &detail.Status,
		&detail.IPPoolID, &detail.IPPoolName, &detail.CreatedAt, &detail.CompanyName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("sub_account %d not found", id)
		}
		return nil, fmt.Errorf("get sub_account: %w", err)
	}

	domains, err := r.domainsForSubAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	detail.Domains = domains
	return &detail, nil
}

func (r *SubAccountRepository) domainsForSubAccount(ctx context.Context, subAccountID int32) ([]DomainLite, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, domain, is_sending, is_blocked
		FROM domain
		WHERE deleted_at IS NULL AND sub_account_id = $1
		ORDER BY domain`, subAccountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []DomainLite
	for rows.Next() {
		var d DomainLite
		if err := rows.Scan(&d.ID, &d.Domain, &d.IsSending, &d.IsBlocked); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

func (r *SubAccountRepository) GetByIDs(ctx context.Context, ids []int32) (map[int32]SubAccount, error) {
	out := map[int32]SubAccount{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT sa.id, sa.company_id, sa.name, sa.status,
		       COALESCE(ip.name, ''),
		       FLOOR(EXTRACT(EPOCH FROM COALESCE(sa.created_at, CURRENT_TIMESTAMP))::bigint * 1000),
		       COALESCE(c.name, '')
		FROM sub_account sa
		LEFT JOIN ip_pool ip ON ip.id = sa.ip_pool_id AND ip.deleted_at IS NULL
		LEFT JOIN company c ON c.id = sa.company_id AND c.deleted_at IS NULL
		WHERE sa.deleted_at IS NULL AND sa.id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("get sub_accounts by ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sa SubAccount
		if err := rows.Scan(&sa.ID, &sa.CompanyID, &sa.Name, &sa.Status, &sa.IPPoolName, &sa.CreatedAt, &sa.CompanyName); err != nil {
			return nil, err
		}
		out[sa.ID] = sa
	}
	return out, rows.Err()
}

func (r *SubAccountRepository) UpdateStatus(ctx context.Context, id int32, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE sub_account
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`, status, id)
	if err != nil {
		return fmt.Errorf("update sub_account status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("sub_account %d not found", id)
	}
	return nil
}
