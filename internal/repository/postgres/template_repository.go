package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	templatedomain "example.com/taskservice/internal/domain/template"
)

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

func (r *TemplateRepository) Create(ctx context.Context, tmpl *templatedomain.TaskTemplate) (*templatedomain.TaskTemplate, error) {
	const query = `
		INSERT INTO task_templates (title, description, rule_type, rule_params, execution_time, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	now := time.Now().UTC()
	err := r.pool.QueryRow(ctx, query,
		tmpl.Title, tmpl.Description, tmpl.RuleType, tmpl.RuleParams, tmpl.ExecutionTime, now, now,
	).Scan(&tmpl.ID, &tmpl.CreatedAt, &tmpl.UpdatedAt)
	if err != nil {
		return nil, err
	}
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now
	return tmpl, nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*templatedomain.TaskTemplate, error) {
	const query = `
		SELECT id, title, description, rule_type, rule_params, execution_time, created_at, updated_at
		FROM task_templates
		WHERE id = $1
	`
	var tmpl templatedomain.TaskTemplate
	var ruleParams []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tmpl.ID, &tmpl.Title, &tmpl.Description, &tmpl.RuleType, &ruleParams,
		&tmpl.ExecutionTime, &tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, templatedomain.ErrNotFound
		}
		return nil, err
	}
	tmpl.RuleParams = json.RawMessage(ruleParams)
	return &tmpl, nil
}

func (r *TemplateRepository) Update(ctx context.Context, tmpl *templatedomain.TaskTemplate) (*templatedomain.TaskTemplate, error) {
	const query = `
		UPDATE task_templates
		SET title = $1, description = $2, rule_type = $3, rule_params = $4, execution_time = $5, updated_at = $6
		WHERE id = $7
		RETURNING updated_at
	`
	now := time.Now().UTC()
	err := r.pool.QueryRow(ctx, query,
		tmpl.Title, tmpl.Description, tmpl.RuleType, tmpl.RuleParams, tmpl.ExecutionTime, now, tmpl.ID,
	).Scan(&tmpl.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, templatedomain.ErrNotFound
		}
		return nil, err
	}
	tmpl.UpdatedAt = now
	return tmpl, nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM task_templates WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return templatedomain.ErrNotFound
	}
	return nil
}

func (r *TemplateRepository) List(ctx context.Context) ([]templatedomain.TaskTemplate, error) {
	const query = `
		SELECT id, title, description, rule_type, rule_params, execution_time, created_at, updated_at
		FROM task_templates
		ORDER BY id DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var templates []templatedomain.TaskTemplate
	for rows.Next() {
		var tmpl templatedomain.TaskTemplate
		var ruleParams []byte
		if err := rows.Scan(&tmpl.ID, &tmpl.Title, &tmpl.Description, &tmpl.RuleType, &ruleParams,
			&tmpl.ExecutionTime, &tmpl.CreatedAt, &tmpl.UpdatedAt); err != nil {
			return nil, err
		}
		tmpl.RuleParams = json.RawMessage(ruleParams)
		templates = append(templates, tmpl)
	}
	return templates, rows.Err()
}
