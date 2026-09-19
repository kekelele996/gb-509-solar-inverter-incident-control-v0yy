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

type SolarSiteService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.SolarSite], error)
	Get(context.Context, uint) (model.SolarSite, error)
	Create(context.Context, dto.CreateSolarSite, string, string) (model.SolarSite, error)
	Update(context.Context, uint, dto.UpdateSolarSite, string, string) (model.SolarSite, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.SolarSite, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type solarSiteService struct {
	repository repository.SolarSiteRepository
	security   SecurityService
}

func NewSolarSiteService(repo repository.SolarSiteRepository, security SecurityService) SolarSiteService {
	return &solarSiteService{repository: repo, security: security}
}

func (s *solarSiteService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.SolarSite], error) {
	return s.repository.List(ctx, query)
}

func (s *solarSiteService) Get(ctx context.Context, id uint) (model.SolarSite, error) {
	return s.repository.Get(ctx, id)
}

func (s *solarSiteService) Create(ctx context.Context, input dto.CreateSolarSite, actor, requestID string) (model.SolarSite, error) {
	if err := validateSolarSiteBusinessFields(input.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.SolarSite{}, err
	}
	item := model.SolarSite{
		BaseModel: model.BaseModel{
			Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name),
			Status: model.SolarSiteInitialStatus, Version: 1, Description: strings.TrimSpace(input.Description),
		},
		Facility: strings.TrimSpace(input.Facility), Owner: strings.TrimSpace(input.Owner),
		Category: strings.TrimSpace(input.Category), RiskLevel: input.RiskLevel,
		MetricValue: input.MetricValue, MetricUnit: strings.TrimSpace(input.MetricUnit),
		EffectiveAt: input.EffectiveAt.UTC(), Evidence: strings.TrimSpace(input.Evidence),
		RelatedCode: strings.ToUpper(strings.TrimSpace(input.RelatedCode)),
	}
	if err := s.repository.Create(ctx, &item); err != nil {
		return model.SolarSite{}, fmt.Errorf("create 光伏场站: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "create", "SolarSite", item.ID, "", item.Status, "created 光伏场站")
	return item, nil
}

func (s *solarSiteService) Update(ctx context.Context, id uint, input dto.UpdateSolarSite, actor, requestID string) (model.SolarSite, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.SolarSite{}, err
	}
	if err := validateSolarSiteBusinessFields(current.Code, input.Name, input.Facility, input.Owner); err != nil {
		return model.SolarSite{}, err
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
		return model.SolarSite{}, fmt.Errorf("update 光伏场站: %w", err)
	}
	_ = s.security.Audit(ctx, actor, requestID, "update", "SolarSite", id, current.Status, current.Status, "updated business fields")
	return s.repository.Get(ctx, id)
}

func (s *solarSiteService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (model.SolarSite, error) {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return model.SolarSite{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.SolarSiteTransitions, current.Status, target) {
		return model.SolarSite{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, id, input.ExpectedVersion, &current); err != nil {
		return model.SolarSite{}, fmt.Errorf("transition 光伏场站: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "SolarSite", id, before, target, input.Reason); err != nil {
		return model.SolarSite{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, id)
}

func (s *solarSiteService) Delete(ctx context.Context, id uint, actor, requestID string) error {
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}
	return s.security.Audit(ctx, actor, requestID, "delete", "SolarSite", id, current.Status, "deleted", "soft deleted 光伏场站")
}

func (s *solarSiteService) StatusCounts(ctx context.Context) (map[string]int64, error) {
	return s.repository.CountByStatus(ctx)
}

func validateSolarSiteBusinessFields(code, name, facility, owner string) error {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(facility) == "" || strings.TrimSpace(owner) == "" {
		return ErrInvalidInput
	}
	return nil
}
