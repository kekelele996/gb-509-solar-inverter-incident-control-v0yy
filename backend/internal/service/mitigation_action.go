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

type MitigationActionService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.MitigationAction], error)
	Get(context.Context, uint) (model.MitigationAction, error)
	Create(context.Context, dto.CreateMitigationAction, string, string) (model.MitigationAction, error)
	Update(context.Context, uint, dto.UpdateMitigationAction, string, string) (model.MitigationAction, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.MitigationAction, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type mitigationActionService struct {
	repository repository.MitigationActionRepository
	security   SecurityService
}

func NewMitigationActionService(repo repository.MitigationActionRepository, security SecurityService) MitigationActionService {
	return &mitigationActionService{repository: repo, security: security}
}

func (s *mitigationActionService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.MitigationAction], error) {
	return s.repository.List(ctx, query)
}

func (s *mitigationActionService) Get(ctx context.Context, id uint) (model.MitigationAction, error) {
	return s.repository.Get(ctx, id)
}

func (s *mitigationActionService) Create(ctx context.Context, input dto.CreateMitigationAction, actor, requestID string) (model.MitigationAction, error) {
	if err := validateMitigationActionBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.MitigationAction{}, err
	}
	item := model.MitigationAction{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.MitigationActionInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.MitigationAction{}, fmt.Errorf("create 处置动作: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "MitigationAction", item.ID, "", item.Status, "created 处置动作")
	return item, nil
}

func (s *mitigationActionService) Update(ctx context.Context, id uint, input dto.UpdateMitigationAction, actor, requestID string) (model.MitigationAction, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.MitigationAction{}, err
	}
	if err := validateMitigationActionBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.MitigationAction{}, err
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
		return model.MitigationAction{}, fmt.Errorf("update 处置动作: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "MitigationAction", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *mitigationActionService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.MitigationAction, error) {
	if err := requireSecondConfirmation(input); err != nil {
		return model.MitigationAction{}, err
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.MitigationAction{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.MitigationActionTransitions, current.Status, target) {
		return model.MitigationAction{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.MitigationAction{}, fmt.Errorf("transition 处置动作: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "MitigationAction", id, before, target, input.Reason); err != nil {
		return model.MitigationAction{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func requireSecondConfirmation(input dto.TransitionRequest) error {
	if !input.Confirmed {
		return fmt.Errorf("%w: remote action requires explicit second confirmation", ErrInvalidInput)
	}
	return nil
}

func (s *mitigationActionService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "MitigationAction", id, current.Status, "deleted", "soft deleted 处置动作")
}

func (s *mitigationActionService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateMitigationActionBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
