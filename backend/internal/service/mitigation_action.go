package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/constants"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/repository"
	"gorm.io/gorm"
)

// ConfirmInterlockResult returns the three aggregates persisted by the
// interlocked confirm so callers can display 关联对象 without extra queries.
type ConfirmInterlockResult struct {
	Action   model.MitigationAction `json:"action"`
	Fault    model.FaultEvent       `json:"fault"`
	Inverter model.InverterUnit     `json:"inverter"`
}

type MitigationActionService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.MitigationAction], error)
	Get(context.Context, uint) (model.MitigationAction, error)
	Create(context.Context, dto.CreateMitigationAction, string, string) (model.MitigationAction, error)
	Update(context.Context, uint, dto.UpdateMitigationAction, string, string) (model.MitigationAction, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (model.MitigationAction, error)
	ConfirmInterlock(context.Context, uint, dto.ConfirmMitigationAction, string, string) (ConfirmInterlockResult, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type mitigationActionService struct {
	db         *gorm.DB
	repository repository.MitigationActionRepository
	faults     repository.FaultEventRepository
	inverters  repository.InverterUnitRepository
	security   SecurityService
}

func NewMitigationActionService(db *gorm.DB, repo repository.MitigationActionRepository, faults repository.FaultEventRepository, inverters repository.InverterUnitRepository, security SecurityService) MitigationActionService {
	return &mitigationActionService{db: db, repository: repo, faults: faults, inverters: inverters, security: security}
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
	if target == "executing" {
		return model.MitigationAction{}, fmt.Errorf("%w: %s -> executing must go through the interlocked confirm endpoint", ErrInvalidTransition, current.Status)
	}
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

// ConfirmInterlock 处置动作确认接入故障联锁: the action is linked to its fault
// event and inverter through 关联编号 + 场站. The confirm only passes when the
// fault is 已确认 (acknowledged) and the inverter is 告警 (warning) or 脱网
// (tripped). On success the action, fault and inverter move to
// executing/mitigated/isolated in a single transaction together with their
// audit rows; any version conflict, state mismatch or write failure rolls all
// of them back, so repeated or concurrent confirms succeed exactly once.
func (s *mitigationActionService) ConfirmInterlock(ctx context.Context, id uint, input dto.ConfirmMitigationAction, actor, requestID string) (ConfirmInterlockResult, error) {
	if !input.Confirmed {
		return ConfirmInterlockResult{}, fmt.Errorf("%w: remote action requires explicit second confirmation", ErrInvalidInput)
	}
	action, err := s.repository.Get(ctx, id)
	if err != nil {
		return ConfirmInterlockResult{}, err
	}
	if !constants.CanTransition(constants.MitigationActionTransitions, action.Status, "executing") {
		return ConfirmInterlockResult{}, fmt.Errorf("%w: %s -> executing", ErrInvalidTransition, action.Status)
	}
	relatedCode := strings.ToUpper(strings.TrimSpace(action.RelatedCode))
	facility := strings.TrimSpace(action.Facility)
	if relatedCode == "" || facility == "" {
		return ConfirmInterlockResult{}, fmt.Errorf("%w: action %s has no related code or facility", ErrInterlockLink, action.Code)
	}
	fault, err := s.faults.FindByLink(ctx, relatedCode, facility)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ConfirmInterlockResult{}, fmt.Errorf("%w: no fault event for %s @ %s", ErrInterlockLink, relatedCode, facility)
		}
		return ConfirmInterlockResult{}, fmt.Errorf("load linked 故障事件: %w", err)
	}
	inverter, err := s.inverters.FindByLink(ctx, relatedCode, facility)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ConfirmInterlockResult{}, fmt.Errorf("%w: no inverter for %s @ %s", ErrInterlockLink, relatedCode, facility)
		}
		return ConfirmInterlockResult{}, fmt.Errorf("load linked 逆变器: %w", err)
	}
	if fault.Status != string(constants.FaultStateAcknowledged) {
		return ConfirmInterlockResult{}, fmt.Errorf("%w: fault %s is %s, must be acknowledged", ErrInterlockPrecondition, fault.Code, fault.Status)
	}
	if inverter.Status != string(constants.InverterStateWarning) && inverter.Status != string(constants.InverterStateTripped) {
		return ConfirmInterlockResult{}, fmt.Errorf("%w: inverter %s is %s, must be warning or tripped", ErrInterlockPrecondition, inverter.Code, inverter.Status)
	}

	now := time.Now().UTC()
	actionBefore, faultBefore, inverterBefore := action.Status, fault.Status, inverter.Status
	faultVersion, inverterVersion := fault.Version, inverter.Version
	err = s.db.Transaction(func(tx *gorm.DB) error {
		actionRepo := repository.NewMitigationActionRepository(tx)
		faultRepo := repository.NewFaultEventRepository(tx)
		inverterRepo := repository.NewInverterUnitRepository(tx)
		auditRepo := repository.NewSecurityRepository(tx)

		action.Status, action.Version, action.UpdatedAt = "executing", action.Version+1, now
		if err := actionRepo.Update(ctx, action.ID, input.ExpectedVersion, &action); err != nil {
			return fmt.Errorf("confirm 处置动作: %w", err)
		}
		fault.Status, fault.Version, fault.UpdatedAt = string(constants.FaultStateMitigated), faultVersion+1, now
		if err := faultRepo.Update(ctx, fault.ID, faultVersion, &fault); err != nil {
			return fmt.Errorf("mitigate linked 故障事件: %w", err)
		}
		inverter.Status, inverter.Version, inverter.UpdatedAt = string(constants.InverterStateIsolated), inverterVersion+1, now
		if err := inverterRepo.Update(ctx, inverter.ID, inverterVersion, &inverter); err != nil {
			return fmt.Errorf("isolate linked 逆变器: %w", err)
		}
		detail := fmt.Sprintf("fault interlock confirm: %s", strings.TrimSpace(input.Reason))
		if err := appendAudit(ctx, auditRepo, actor, requestID, "confirm", "MitigationAction", action.ID, actionBefore, action.Status, detail); err != nil {
			return err
		}
		if err := appendAudit(ctx, auditRepo, actor, requestID, "confirm", "FaultEvent", fault.ID, faultBefore, fault.Status, detail); err != nil {
			return err
		}
		return appendAudit(ctx, auditRepo, actor, requestID, "confirm", "InverterUnit", inverter.ID, inverterBefore, inverter.Status, detail)
	})
	if err != nil {
		return ConfirmInterlockResult{}, err
	}
	return s.readBackInterlock(ctx, action.ID, fault.ID, inverter.ID)
}

// readBackInterlock re-reads the three aggregates after commit so the response
// always reflects the persisted state a page refresh would return.
func (s *mitigationActionService) readBackInterlock(ctx context.Context, actionID, faultID, inverterID uint) (ConfirmInterlockResult, error) {
	action, err := s.repository.Get(ctx, actionID)
	if err != nil {
		return ConfirmInterlockResult{}, err
	}
	fault, err := s.faults.Get(ctx, faultID)
	if err != nil {
		return ConfirmInterlockResult{}, err
	}
	inverter, err := s.inverters.Get(ctx, inverterID)
	if err != nil {
		return ConfirmInterlockResult{}, err
	}
	return ConfirmInterlockResult{Action: action, Fault: fault, Inverter: inverter}, nil
}

// appendAudit mirrors SecurityService.Audit defaulting but writes through the
// transaction-scoped repository so the audit row commits or rolls back
// together with the interlocked state changes.
func appendAudit(ctx context.Context, repo repository.SecurityRepository, actor, requestID, action, entityType string, entityID uint, before, after, detail string) error {
	if actor == "" {
		actor = "system"
	}
	if requestID == "" {
		requestID = "untracked"
	}
	return repo.AppendAudit(ctx, &model.AuditLog{
		Actor: actor, RequestID: requestID, Action: action, EntityType: entityType,
		EntityID: entityID, BeforeState: before, AfterState: after, Detail: detail,
		CreatedAt: time.Now().UTC(),
	})
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
