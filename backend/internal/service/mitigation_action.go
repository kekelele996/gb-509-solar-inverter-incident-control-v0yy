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

type MitigationActionService interface {
	List(context.Context, dto.PageQuery) (repository.Page[model.MitigationAction], error)
	Get(context.Context, uint) (model.MitigationAction, error)
	Interlock(context.Context, uint) (dto.ActionInterlockView, error)
	Create(context.Context, dto.CreateMitigationAction, string, string) (model.MitigationAction, error)
	Update(context.Context, uint, dto.UpdateMitigationAction, string, string) (model.MitigationAction, error)
	Transition(context.Context, uint, dto.TransitionRequest, string, string) (dto.ActionInterlockView, error)
	Delete(context.Context, uint, string, string) error
	StatusCounts(context.Context) (map[string]int64, error)
}

type mitigationActionService struct {
	repository repository.MitigationActionRepository
	faults     repository.FaultEventRepository
	inverters  repository.InverterUnitRepository
	tx         repository.TransactionManager
	security   SecurityService
}

func NewMitigationActionService(
	repo repository.MitigationActionRepository,
	faults repository.FaultEventRepository,
	inverters repository.InverterUnitRepository,
	tx repository.TransactionManager,
	security SecurityService,
) MitigationActionService {
	return &mitigationActionService{repository: repo, faults: faults, inverters: inverters, tx: tx, security: security}
}

func (s *mitigationActionService) List(ctx context.Context, query dto.PageQuery) (repository.Page[model.MitigationAction], error) {
	return s.repository.List(ctx, query)
}

func (s *mitigationActionService) Get(ctx context.Context, id uint) (model.MitigationAction, error) {
	return s.repository.Get(ctx, id)
}

func (s *mitigationActionService) Interlock(ctx context.Context, id uint) (dto.ActionInterlockView, error) {
	action, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.ActionInterlockView{}, err
	}
	fault, inverter, reason := s.loadInterlockState(ctx, action, true, action.Status == "executing")
	return dto.ActionInterlockView{
		Action:        action,
		Fault:         fault,
		Inverter:      inverter,
		FailureReason: reason,
		CanConfirm:    reason == "" && action.Status == "confirmed" && fault != nil && inverter != nil,
	}, nil
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

func (s *mitigationActionService) Transition(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (dto.ActionInterlockView, error) {
	if err := requireSecondConfirmation(input); err != nil {
		return dto.ActionInterlockView{}, err
	}
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return dto.ActionInterlockView{}, err
	}
	target := strings.TrimSpace(input.Status)
	if !constants.CanTransition(constants.MitigationActionTransitions, current.Status, target) {
		return dto.ActionInterlockView{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, current.Status, target)
	}
	if target == "executing" {
		return s.executeInterlock(ctx, id, input, actor, requestID)
	}
	updated, err := s.simpleTransition(ctx, current, target, input, actor, requestID)
	if err != nil {
		return dto.ActionInterlockView{}, err
	}
	return dto.ActionInterlockView{Action: updated, CanConfirm: false}, nil
}

func (s *mitigationActionService) simpleTransition(ctx context.Context, current model.MitigationAction, target string, input dto.TransitionRequest, actor, requestID string) (model.MitigationAction, error) {
	before := current.Status
	current.Status = target
	current.Version = input.ExpectedVersion + 1
	current.UpdatedAt = time.Now().UTC()
	if err := s.repository.Update(ctx, current.ID, input.ExpectedVersion, &current); err != nil {
		return model.MitigationAction{}, fmt.Errorf("transition 处置动作: %w", err)
	}
	if err := s.security.Audit(ctx, actor, requestID, "transition", "MitigationAction", current.ID, before, target, input.Reason); err != nil {
		return model.MitigationAction{}, fmt.Errorf("persist transition audit: %w", err)
	}
	return s.repository.Get(ctx, current.ID)
}

func (s *mitigationActionService) executeInterlock(ctx context.Context, id uint, input dto.TransitionRequest, actor, requestID string) (dto.ActionInterlockView, error) {
	if err := s.tx.InTx(ctx, func(txCtx context.Context) error {
		action, err := s.repository.Get(txCtx, id)
		if err != nil {
			return err
		}
		if !constants.CanTransition(constants.MitigationActionTransitions, action.Status, "executing") {
			return fmt.Errorf("%w: %s -> executing", ErrInvalidTransition, action.Status)
		}
		if action.Version != input.ExpectedVersion {
			return repository.ErrVersionConflict
		}

		fault, inverter, reason := s.loadInterlockState(txCtx, action, false, false)
		if reason != "" {
			return fmt.Errorf("%w: %s", ErrInvalidTransition, reason)
		}
		if err := s.repository.UpdateStatus(txCtx, action.ID, action.Version, "executing"); err != nil {
			return fmt.Errorf("transition 处置动作: %w", err)
		}
		if err := s.faults.UpdateStatus(txCtx, fault.ID, fault.Version, string(constants.FaultStateMitigated)); err != nil {
			return fmt.Errorf("transition 故障事件: %w", err)
		}
		if err := s.inverters.UpdateStatus(txCtx, inverter.ID, inverter.Version, string(constants.InverterStateIsolated)); err != nil {
			return fmt.Errorf("transition 逆变器: %w", err)
		}
		if err := s.security.Audit(txCtx, actor, requestID, "transition", "MitigationAction", action.ID, action.Status, "executing", input.Reason); err != nil {
			return fmt.Errorf("persist action audit: %w", err)
		}
		if err := s.security.Audit(txCtx, actor, requestID, "interlock", "FaultEvent", fault.ID, fault.Status, "mitigated", "故障联锁：处置动作确认"); err != nil {
			return fmt.Errorf("persist fault audit: %w", err)
		}
		if err := s.security.Audit(txCtx, actor, requestID, "interlock", "InverterUnit", inverter.ID, inverter.Status, "isolated", "故障联锁：逆变器隔离"); err != nil {
			return fmt.Errorf("persist inverter audit: %w", err)
		}
		return nil
	}); err != nil {
		return dto.ActionInterlockView{}, err
	}
	return s.Interlock(ctx, id)
}

func (s *mitigationActionService) loadInterlockState(ctx context.Context, action model.MitigationAction, includeActionStatus, committed bool) (*model.FaultEvent, *model.InverterUnit, string) {
	relatedCode := strings.ToUpper(strings.TrimSpace(action.RelatedCode))
	facility := strings.TrimSpace(action.Facility)
	if relatedCode == "" {
		return nil, nil, "处置动作缺少关联编号，无法定位故障事件"
	}

	fault, faultErr := s.faults.GetByRelatedCodeAndFacility(ctx, relatedCode, facility)
	inverter, inverterErr := s.inverters.GetByRelatedCodeAndFacility(ctx, relatedCode, facility)
	reasons := make([]string, 0, 3)
	var faultPtr *model.FaultEvent
	var inverterPtr *model.InverterUnit

	switch {
	case errors.Is(faultErr, gorm.ErrRecordNotFound):
		reasons = append(reasons, fmt.Sprintf("场站 %q 中未找到关联编号 %q 的故障事件", facility, relatedCode))
	case faultErr != nil:
		reasons = append(reasons, "联锁故障事件读取失败，请稍后重试")
	default:
		faultPtr = &fault
		if !committed && fault.Status != string(constants.FaultStateAcknowledged) {
			reasons = append(reasons, fmt.Sprintf("故障事件 %s 当前为 %s，需处于已确认状态", fault.Code, fault.Status))
		}
	}

	switch {
	case errors.Is(inverterErr, gorm.ErrRecordNotFound):
		reasons = append(reasons, fmt.Sprintf("场站 %q 中未找到关联编号 %q 的逆变器", facility, relatedCode))
	case inverterErr != nil:
		reasons = append(reasons, "联锁逆变器读取失败，请稍后重试")
	default:
		inverterPtr = &inverter
		if committed {
			if inverter.Status != string(constants.InverterStateIsolated) {
				reasons = append(reasons, fmt.Sprintf("逆变器 %s 当前为 %s，联锁完成后应为隔离状态", inverter.Code, inverter.Status))
			}
		} else if inverter.Status != string(constants.InverterStateWarning) && inverter.Status != string(constants.InverterStateTripped) {
			reasons = append(reasons, fmt.Sprintf("逆变器 %s 当前为 %s，需处于告警或脱网状态", inverter.Code, inverter.Status))
		}
	}

	if includeActionStatus {
		if committed {
			if action.Status != "executing" {
				reasons = append(reasons, fmt.Sprintf("处置动作 %s 当前为 %s，联锁完成后应为执行中", action.Code, action.Status))
			}
		} else if action.Status != "confirmed" {
			reasons = append(reasons, fmt.Sprintf("处置动作 %s 当前为 %s，需处于已确认状态", action.Code, action.Status))
		}
	}
	return faultPtr, inverterPtr, strings.Join(reasons, "；")
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
