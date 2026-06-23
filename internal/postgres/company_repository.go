package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

type CompanyRepository struct {
	pool *pgxpool.Pool
}

func NewCompanyRepository(pool *pgxpool.Pool) *CompanyRepository {
	return &CompanyRepository{pool: pool}
}

func (r *CompanyRepository) GetByID(ctx context.Context, id int32) (*Company, error) {
	var c Company
	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(name, ''), COALESCE(active, false)
		FROM company
		WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&c.ID, &c.Name, &c.Active)
	if err != nil {
		return nil, fmt.Errorf("get company %d: %w", id, err)
	}
	return &c, nil
}

func (r *CompanyRepository) GetOwnerByCompanyID(ctx context.Context, companyID int32) (*CompanyOwner, error) {
	var owner CompanyOwner
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(u.email, ''),
		       COALESCE(
		           NULLIF(TRIM(u.fullname), ''),
		           NULLIF(TRIM(CONCAT(COALESCE(u.firstname, ''), ' ', COALESCE(u.lastname, ''))), ''),
		           u.email
		       )
		FROM company c
		JOIN "user" u ON u.id = c.owner_id AND u.deleted_at IS NULL
		WHERE c.id = $1
		  AND c.deleted_at IS NULL
		  AND u.active = true`, companyID,
	).Scan(&owner.Email, &owner.Name)
	if err != nil {
		return nil, fmt.Errorf("get company %d owner: %w", companyID, err)
	}
	if owner.Email == "" {
		return nil, fmt.Errorf("company %d owner has no email", companyID)
	}
	return &owner, nil
}

func (r *CompanyRepository) GetByIDs(ctx context.Context, ids []int32) (map[int32]Company, error) {
	out := map[int32]Company{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, COALESCE(name, ''), COALESCE(active, false)
		FROM company
		WHERE deleted_at IS NULL AND id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("get companies by ids: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Active); err != nil {
			return nil, err
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}

type CompanyListFilter struct {
	CompanyID *int32
	Search    string
	Page      int
	Size      int
}

func (r *CompanyRepository) List(ctx context.Context, f CompanyListFilter) ([]Company, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size <= 0 {
		f.Size = 20
	}
	offset := (f.Page - 1) * f.Size

	where := []string{"deleted_at IS NULL"}
	args := []interface{}{}
	argN := 1

	if f.CompanyID != nil {
		where = append(where, fmt.Sprintf("id = $%d", argN))
		args = append(args, *f.CompanyID)
		argN++
	}
	if f.Search != "" {
		where = append(where, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+f.Search+"%")
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM company WHERE %s", whereSQL)
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count companies: %w", err)
	}

	listArgs := append(args, f.Size, offset)
	query := fmt.Sprintf(`
		SELECT id, COALESCE(name, ''), COALESCE(active, false)
		FROM company
		WHERE %s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d`, whereSQL, argN, argN+1)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	var list []Company
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Active); err != nil {
			return nil, 0, err
		}
		list = append(list, c)
	}
	return list, total, rows.Err()
}
