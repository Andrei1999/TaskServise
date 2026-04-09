package template

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	templatedomain "example.com/taskservice/internal/domain/template"
)

const moscowTZ = "Europe/Moscow"

type Service struct {
	templateRepo TemplateRepository
	taskRepo     TaskRepository
	now          func() time.Time
	loc          *time.Location
}

func NewService(templateRepo TemplateRepository, taskRepo TaskRepository) *Service {
	loc, err := time.LoadLocation(moscowTZ)
	if err != nil {
		log.Printf("Warning: failed to load Moscow timezone: %v, using UTC", err)
		loc = time.UTC
	}
	
	return &Service{
		templateRepo: templateRepo,
		taskRepo:     taskRepo,
		now: func() time.Time {
			return time.Now().In(loc)
		},
		loc: loc,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*templatedomain.TaskTemplate, error) {
	if err := validateCreateInput(input); err != nil {
		return nil, err
	}
	
	ruleParamsJSON, err := json.Marshal(input.RuleParams)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid rule params", templatedomain.ErrInvalidInput)
	}
	
	tmpl := &templatedomain.TaskTemplate{
		Title:         input.Title,
		Description:   input.Description,
		RuleType:      input.RuleType,
		RuleParams:    ruleParamsJSON,
		ExecutionTime: input.ExecutionTime,
	}
	
	created, err := s.templateRepo.Create(ctx, tmpl)
	if err != nil {
		return nil, err
	}
	
	// Генерация задач на месяц вперёд
	now := s.now()
	endDate := now.AddDate(0, 1, 0)
	
	log.Printf("Generating tasks for template %d from %s to %s", created.ID, now.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	if err := s.generateInstances(ctx, created, now, endDate); err != nil {
		log.Printf("Error generating instances: %v", err)
		return nil, fmt.Errorf("failed to generate instances: %w", err)
	}
	
	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*templatedomain.TaskTemplate, error) {
	return s.templateRepo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*templatedomain.TaskTemplate, error) {
	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// удаляем будущие экземпляры со статусом 'new'
	if err := s.deleteFutureNewInstances(ctx, id); err != nil {
		return nil, err
	}
	
	if input.Title != nil {
		existing.Title = *input.Title
	}
	if input.Description != nil {
		existing.Description = *input.Description
	}
	if input.RuleType != nil {
		existing.RuleType = *input.RuleType
	}
	if input.RuleParams != nil {
		paramsJSON, err := json.Marshal(input.RuleParams)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid rule params", templatedomain.ErrInvalidInput)
		}
		existing.RuleParams = paramsJSON
	}
	if input.ExecutionTime != nil {
		existing.ExecutionTime = *input.ExecutionTime
	}
	
	if err := validateTemplate(existing); err != nil {
		return nil, err
	}
	
	updated, err := s.templateRepo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	
	// перегенерация на месяц вперёд
	now := s.now()
	endDate := now.AddDate(0, 1, 0)
	
	if err := s.generateInstances(ctx, updated, now, endDate); err != nil {
		log.Printf("Error generating instances during update: %v", err)
		return nil, fmt.Errorf("failed to generate instances: %w", err)
	}
	
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.templateRepo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]templatedomain.TaskTemplate, error) {
	return s.templateRepo.List(ctx)
}

func (s *Service) GenerateMissingInstances(ctx context.Context) error {
	templates, err := s.templateRepo.List(ctx)
	if err != nil {
		return err
	}
	
	now := s.now()
	for _, tmpl := range templates {
		lastDate, err := s.getLastGeneratedDate(ctx, tmpl.ID)
		if err != nil {
			log.Printf("Error getting last generated date for template %d: %v", tmpl.ID, err)
			continue
		}
		
		start := lastDate.AddDate(0, 0, 1)
		end := start.AddDate(0, 1, 0)
		
		if start.Before(now) {
			start = now
		}
		
		if err := s.generateInstances(ctx, &tmpl, start, end); err != nil {
			log.Printf("Error generating instances for template %d: %v", tmpl.ID, err)
		}
	}
	return nil
}

func (s *Service) deleteFutureNewInstances(ctx context.Context, templateID int64) error {
	tasks, err := s.taskRepo.ListByTemplate(ctx, templateID)
	if err != nil {
		return err
	}
	
	now := s.now()
	for _, t := range tasks {
		if t.ScheduledAt != nil && t.ScheduledAt.After(now) && t.Status == taskdomain.StatusNew {
			if err := s.taskRepo.Delete(ctx, t.ID); err != nil {
				log.Printf("Error deleting task %d: %v", t.ID, err)
			}
		}
	}
	return nil
}

func (s *Service) getLastGeneratedDate(ctx context.Context, templateID int64) (time.Time, error) {
	tasks, err := s.taskRepo.ListByTemplate(ctx, templateID)
	if err != nil {
		return time.Time{}, err
	}
	
	var maxDate time.Time
	for _, t := range tasks {
		if t.ScheduledAt != nil && t.ScheduledAt.After(maxDate) {
			maxDate = *t.ScheduledAt
		}
	}
	
	if maxDate.IsZero() {
		return s.now(), nil
	}
	return maxDate, nil
}

func (s *Service) generateInstances(ctx context.Context, tmpl *templatedomain.TaskTemplate, from, to time.Time) error {
	rule, err := templatedomain.ParseRule(tmpl.RuleType, tmpl.RuleParams)
	if err != nil {
		return fmt.Errorf("parse rule failed: %w", err)
	}
	
	execTime, err := time.Parse("15:04:05", tmpl.ExecutionTime)
	if err != nil {
		return fmt.Errorf("parse execution time failed: %w", err)
	}
	
	count := 0
	for d := from; d.Before(to); d = d.AddDate(0, 0, 1) {
		if s.matchesRule(rule, d, tmpl.RuleType, tmpl.RuleParams, from) {
			scheduled := time.Date(d.Year(), d.Month(), d.Day(), execTime.Hour(), execTime.Minute(), execTime.Second(), 0, s.loc)
			task := &taskdomain.Task{
				Title:       tmpl.Title,
				Description: tmpl.Description,
				Status:      taskdomain.StatusNew,
				CreatedAt:   s.now(),
				UpdatedAt:   s.now(),
				ScheduledAt: &scheduled,
				TemplateID:  &tmpl.ID,
			}
			if _, err := s.taskRepo.Create(ctx, task); err != nil {
				return fmt.Errorf("create task failed: %w", err)
			}
			count++
		}
	}
	
	log.Printf("Generated %d instances for template %d", count, tmpl.ID)
	return nil
}

func (s *Service) matchesRule(rule templatedomain.Rule, date time.Time, ruleType templatedomain.RuleType, params json.RawMessage, startDate time.Time) bool {
	switch ruleType {
	case templatedomain.RuleDaily:
		var r templatedomain.DailyRule
		if err := json.Unmarshal(params, &r); err != nil {
			return false
		}
		daysDiff := int(date.Sub(startDate).Hours() / 24)
		return daysDiff >= 0 && daysDiff%r.IntervalDays == 0
		
	case templatedomain.RuleMonthly:
		var r templatedomain.MonthlyRule
		if err := json.Unmarshal(params, &r); err != nil {
			return false
		}
		return date.Day() == r.DayOfMonth
		
	case templatedomain.RuleSpecificDates:
		var r templatedomain.SpecificDatesRule
		if err := json.Unmarshal(params, &r); err != nil {
			return false
		}
		dateStr := date.Format("2006-01-02")
		for _, d := range r.Dates {
			if d == dateStr {
				return true
			}
		}
		return false
		
	case templatedomain.RuleEvenOdd:
		var r templatedomain.EvenOddRule
		if err := json.Unmarshal(params, &r); err != nil {
			return false
		}
		day := date.Day()
		if r.Parity == "even" {
			return day%2 == 0
		}
		return day%2 == 1
		
	default:
		return false
	}
}

func validateCreateInput(input CreateInput) error {
	if input.Title == "" {
		return fmt.Errorf("%w: title required", templatedomain.ErrInvalidInput)
	}
	if _, err := time.Parse("15:04:05", input.ExecutionTime); err != nil {
		return fmt.Errorf("%w: invalid execution_time", templatedomain.ErrInvalidInput)
	}
	return validateRule(input.RuleType, input.RuleParams)
}

func validateTemplate(tmpl *templatedomain.TaskTemplate) error {
	if tmpl.Title == "" {
		return fmt.Errorf("%w: title required", templatedomain.ErrInvalidInput)
	}
	if _, err := time.Parse("15:04:05", tmpl.ExecutionTime); err != nil {
		return fmt.Errorf("%w: invalid execution_time", templatedomain.ErrInvalidInput)
	}
	var params map[string]interface{}
	if err := json.Unmarshal(tmpl.RuleParams, &params); err != nil {
		return fmt.Errorf("%w: invalid rule_params JSON", templatedomain.ErrInvalidInput)
	}
	return validateRule(tmpl.RuleType, params)
}

func validateRule(ruleType templatedomain.RuleType, params map[string]interface{}) error {
	switch ruleType {
	case templatedomain.RuleDaily:
		interval, ok := params["interval_days"]
		if !ok {
			return fmt.Errorf("%w: daily needs interval_days", templatedomain.ErrInvalidInput)
		}
		val, ok := interval.(float64)
		if !ok || val < 1 {
			return fmt.Errorf("%w: interval_days must be >=1", templatedomain.ErrInvalidInput)
		}
	case templatedomain.RuleMonthly:
		day, ok := params["day_of_month"]
		if !ok {
			return fmt.Errorf("%w: monthly needs day_of_month", templatedomain.ErrInvalidInput)
		}
		val, ok := day.(float64)
		if !ok || val < 1 || val > 31 {
			return fmt.Errorf("%w: day_of_month must be 1..31", templatedomain.ErrInvalidInput)
		}
	case templatedomain.RuleSpecificDates:
		dates, ok := params["dates"]
		if !ok {
			return fmt.Errorf("%w: specific_dates needs dates array", templatedomain.ErrInvalidInput)
		}
		if _, ok := dates.([]interface{}); !ok {
			return fmt.Errorf("%w: dates must be an array", templatedomain.ErrInvalidInput)
		}
	case templatedomain.RuleEvenOdd:
		parity, ok := params["parity"]
		if !ok {
			return fmt.Errorf("%w: even_odd needs parity", templatedomain.ErrInvalidInput)
		}
		if parity != "even" && parity != "odd" {
			return fmt.Errorf("%w: parity must be 'even' or 'odd'", templatedomain.ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unknown rule_type", templatedomain.ErrInvalidInput)
	}
	return nil
}