package handlers

import (
	"encoding/json"
	"time"

	templatedomain "example.com/taskservice/internal/domain/template"
)

type createTemplateRequest struct {
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	RuleType      templatedomain.RuleType `json:"rule_type"`
	RuleParams    map[string]interface{} `json:"rule_params"`
	ExecutionTime string                 `json:"execution_time"` // "15:04:05"
}

type updateTemplateRequest struct {
	Title         *string                 `json:"title,omitempty"`
	Description   *string                 `json:"description,omitempty"`
	RuleType      *templatedomain.RuleType `json:"rule_type,omitempty"`
	RuleParams    *map[string]interface{} `json:"rule_params,omitempty"`
	ExecutionTime *string                 `json:"execution_time,omitempty"`
}

type templateResponse struct {
	ID            int64           `json:"id"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	RuleType      templatedomain.RuleType `json:"rule_type"`
	RuleParams    json.RawMessage  `json:"rule_params"`
	ExecutionTime string          `json:"execution_time"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

func newTemplateResponse(tmpl *templatedomain.TaskTemplate) templateResponse {
	return templateResponse{
		ID:            tmpl.ID,
		Title:         tmpl.Title,
		Description:   tmpl.Description,
		RuleType:      tmpl.RuleType,
		RuleParams:    tmpl.RuleParams,
		ExecutionTime: tmpl.ExecutionTime,
		CreatedAt:     tmpl.CreatedAt,
		UpdatedAt:     tmpl.UpdatedAt,
	}
}
