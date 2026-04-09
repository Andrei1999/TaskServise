package template

import (
	"encoding/json"
	"time"
)

type RuleType string

const (
	RuleDaily        RuleType = "daily"
	RuleMonthly      RuleType = "monthly"
	RuleSpecificDates RuleType = "specific_dates"
	RuleEvenOdd      RuleType = "even_odd"
)

type TaskTemplate struct {
	ID            int64           `json:"id"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	RuleType      RuleType        `json:"rule_type"`
	RuleParams    json.RawMessage `json:"rule_params"`
	ExecutionTime string          `json:"execution_time"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
