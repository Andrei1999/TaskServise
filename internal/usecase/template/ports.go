package template

import (
	"context"

	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/template"
)

type TemplateRepository interface {
	Create(ctx context.Context, tmpl *templatedomain.TaskTemplate) (*templatedomain.TaskTemplate, error)
	GetByID(ctx context.Context, id int64) (*templatedomain.TaskTemplate, error)
	Update(ctx context.Context, tmpl *templatedomain.TaskTemplate) (*templatedomain.TaskTemplate, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]templatedomain.TaskTemplate, error)
}

type TaskRepository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	ListByTemplate(ctx context.Context, templateID int64) ([]taskdomain.Task, error) // новый метод
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*templatedomain.TaskTemplate, error)
	GetByID(ctx context.Context, id int64) (*templatedomain.TaskTemplate, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*templatedomain.TaskTemplate, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]templatedomain.TaskTemplate, error)
	GenerateMissingInstances(ctx context.Context) error // для cron
}

type CreateInput struct {
	Title         string
	Description   string
	RuleType      templatedomain.RuleType
	RuleParams    map[string]interface{}
	ExecutionTime string // "15:04:05"
}

type UpdateInput struct {
	Title         *string
	Description   *string
	RuleType      *templatedomain.RuleType
	RuleParams    *map[string]interface{}
	ExecutionTime *string
}
