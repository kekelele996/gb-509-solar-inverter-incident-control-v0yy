package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/constants"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/repository"
)

type InverterUnitService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.InverterUnit], error)
	Get(context.Context, uint) (model.InverterUnit, error)
	Create(context.Context, dto.CreateInverterUnit, string, string) (model.InverterUnit, error)
	Update(context.Context, uint, dto.UpdateInverterUnit, string, string) (model.InverterUnit, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.InverterUnit, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type inverterUnitService struct {
	repository repository.InverterUnitRepository
	security   SecurityService
}

func NewInverterUnitService(repo repository.InverterUnitRepository, security SecurityService) InverterUnitService {
	return &inverterUnitService{repository: repo, security: security}
}

func (s *inverterUnitService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.InverterUnit], error) {
	return s.repository.List(ctx, query)
}

func (s *inverterUnitService) Get(ctx context.Context, id uint) (model.InverterUnit, error) {
	return s.repository.Get(ctx, id)
}

func (s *inverterUnitService) Create(ctx context.Context, input dto.CreateInverterUnit, actor, requestID string) (model.InverterUnit, error) {
	if err := validateInverterUnitBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.InverterUnit{}, err
	}
	item := model.InverterUnit{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.InverterUnitInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.InverterUnit{}, fmt.Errorf("create 逆变器: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "InverterUnit", item.ID, "", item.Status, "created 逆变器")
	return item, nil
}

func (s *inverterUnitService) Update(ctx context.Context, id uint, input dto.UpdateInverterUnit, actor, requestID string) (model.InverterUnit, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.InverterUnit{}, err
	}
	if err := validateInverterUnitBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.InverterUnit{}, err
	}
	current.Name = strings.TrimSpace(input.Name)
	current.Description = strings.TrimSpace(input.Description)
	current.Facility = strings.TrimSpace(input.Facility)
	current.Owner = strings.TrimSpace(input.Owner)
	current.Category = strings.TrimSpace(input.Category)
	current.RiskLevel = input.RiskLevel
	current.MetricValue = input.MetricValue
	current.MetricUnit = strings.TrimSpace(input.MetricUnit)
	current.EffectiveAt = input.EffectiveAt.UTC()
	current.Evidence = strings.TrimSpace(input.Evidence)
	current.RelatedCode = strings.ToUpper(strings.TrimSpace(input.RelatedCode))
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.InverterUnit{}, fmt.Errorf("update 逆变器: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "InverterUnit", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *inverterUnitService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.InverterUnit, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.InverterUnit{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.InverterUnitTransitions, current.Status, target) {
		return model.InverterUnit{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.InverterUnit{}, fmt.Errorf("transition 逆变器: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "InverterUnit", id, before, target, input.Reason); err != nil {
		return model.InverterUnit{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *inverterUnitService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "InverterUnit", id, current.Status, "deleted", "soft deleted 逆变器")
}

func (s *inverterUnitService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateInverterUnitBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
