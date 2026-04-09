package template

import (
	"encoding/json"
	"fmt"
	"time"
)

type Rule interface {
	IsMatch(date time.Time) bool
}

type DailyRule struct {
	IntervalDays int `json:"interval_days"`
}

func (r DailyRule) IsMatch(date time.Time) bool {
	// Здесь нужна начальная дата, но для упрощения оставим заглушку.
	// В реальном сервисе вы передадите startDate отдельно.
	return true
}

type MonthlyRule struct {
	DayOfMonth int `json:"day_of_month"`
}

func (r MonthlyRule) IsMatch(date time.Time) bool {
	return date.Day() == r.DayOfMonth
}

type SpecificDatesRule struct {
	Dates []string `json:"dates"`
}

func (r SpecificDatesRule) IsMatch(date time.Time) bool {
	for _, d := range r.Dates {
		if d == date.Format("2006-01-02") {
			return true
		}
	}
	return false
}

type EvenOddRule struct {
	Parity string `json:"parity"` // "even" или "odd"
}

func (r EvenOddRule) IsMatch(date time.Time) bool {
	day := date.Day()
	if r.Parity == "even" {
		return day%2 == 0
	}
	return day%2 == 1
}

func ParseRule(ruleType RuleType, params json.RawMessage) (Rule, error) {
	switch ruleType {
	case RuleDaily:
		var r DailyRule
		if err := json.Unmarshal(params, &r); err != nil {
			return nil, fmt.Errorf("%w: daily rule", ErrInvalidRule)
		}
		if r.IntervalDays < 1 {
			return nil, fmt.Errorf("%w: interval_days must be >=1", ErrInvalidRule)
		}
		return r, nil
	case RuleMonthly:
		var r MonthlyRule
		if err := json.Unmarshal(params, &r); err != nil {
			return nil, fmt.Errorf("%w: monthly rule", ErrInvalidRule)
		}
		if r.DayOfMonth < 1 || r.DayOfMonth > 31 {
			return nil, fmt.Errorf("%w: day_of_month must be 1..31", ErrInvalidRule)
		}
		return r, nil
	case RuleSpecificDates:
		var r SpecificDatesRule
		if err := json.Unmarshal(params, &r); err != nil {
			return nil, fmt.Errorf("%w: specific_dates rule", ErrInvalidRule)
		}
		return r, nil
	case RuleEvenOdd:
		var r EvenOddRule
		if err := json.Unmarshal(params, &r); err != nil {
			return nil, fmt.Errorf("%w: even_odd rule", ErrInvalidRule)
		}
		if r.Parity != "even" && r.Parity != "odd" {
			return nil, fmt.Errorf("%w: parity must be 'even' or 'odd'", ErrInvalidRule)
		}
		return r, nil
	default:
		return nil, fmt.Errorf("%w: unknown rule type", ErrInvalidRule)
	}
}