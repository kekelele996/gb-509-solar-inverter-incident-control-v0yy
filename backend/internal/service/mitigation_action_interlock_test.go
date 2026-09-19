package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/config"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/repository"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type interlockFixture struct {
	db       *gorm.DB
	service  MitigationActionService
	action   model.MitigationAction
	fault    model.FaultEvent
	inverter model.InverterUnit
}

// newInterlockFixture builds a fully wired service on in-memory SQLite with one
// linked triple: action + fault + inverter sharing relatedCode and facility.
func newInterlockFixture(t *testing.T, actionStatus, faultStatus, inverterStatus string) *interlockFixture {
	t.Helper()
	dsn := fmt.Sprintf("file:interlock-%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", " ", "-").Replace(t.Name()))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.AuditLog{}, &model.MitigationAction{}, &model.FaultEvent{}, &model.InverterUnit{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now().UTC()
	action := model.MitigationAction{
		BaseModel: model.BaseModel{Code: "MA-LOCK", Name: "联锁确认动作", Status: actionStatus, Version: 1},
		Facility:  "联锁场站", Owner: "运行一组", Category: "常规", RiskLevel: "high",
		EffectiveAt: now, RelatedCode: "REL-LOCK",
	}
	fault := model.FaultEvent{
		BaseModel: model.BaseModel{Code: "FE-LOCK", Name: "联锁故障", Status: faultStatus, Version: 1},
		Facility:  "联锁场站", Owner: "运行一组", Category: "常规", RiskLevel: "high",
		EffectiveAt: now, RelatedCode: "REL-LOCK",
	}
	inverter := model.InverterUnit{
		BaseModel: model.BaseModel{Code: "IU-LOCK", Name: "联锁逆变器", Status: inverterStatus, Version: 1},
		Facility:  "联锁场站", Owner: "运行一组", Category: "常规", RiskLevel: "high",
		EffectiveAt: now, RelatedCode: "REL-LOCK",
	}
	if err := db.Create(&action).Error; err != nil {
		t.Fatalf("seed action: %v", err)
	}
	if err := db.Create(&fault).Error; err != nil {
		t.Fatalf("seed fault: %v", err)
	}
	if err := db.Create(&inverter).Error; err != nil {
		t.Fatalf("seed inverter: %v", err)
	}
	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	service := NewMitigationActionService(db,
		repository.NewMitigationActionRepository(db),
		repository.NewFaultEventRepository(db),
		repository.NewInverterUnitRepository(db),
		security,
	)
	return &interlockFixture{db: db, service: service, action: action, fault: fault, inverter: inverter}
}

func (f *interlockFixture) confirmInput() dto.ConfirmMitigationAction {
	return dto.ConfirmMitigationAction{ExpectedVersion: f.action.Version, Reason: "现场复核完成，执行联锁确认", Confirmed: true}
}

func (f *interlockFixture) states(t *testing.T) (string, string, string) {
	t.Helper()
	var action model.MitigationAction
	var fault model.FaultEvent
	var inverter model.InverterUnit
	if err := f.db.First(&action, f.action.ID).Error; err != nil {
		t.Fatalf("reload action: %v", err)
	}
	if err := f.db.First(&fault, f.fault.ID).Error; err != nil {
		t.Fatalf("reload fault: %v", err)
	}
	if err := f.db.First(&inverter, f.inverter.ID).Error; err != nil {
		t.Fatalf("reload inverter: %v", err)
	}
	return action.Status, fault.Status, inverter.Status
}

func (f *interlockFixture) auditCount(t *testing.T) int64 {
	t.Helper()
	var total int64
	if err := f.db.Model(&model.AuditLog{}).Count(&total).Error; err != nil {
		t.Fatalf("count audits: %v", err)
	}
	return total
}

func TestConfirmInterlockPersistsTripleAtomically(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	result, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-1")
	if err != nil {
		t.Fatalf("confirm interlock: %v", err)
	}
	if result.Action.Status != "executing" || result.Fault.Status != "mitigated" || result.Inverter.Status != "isolated" {
		t.Fatalf("unexpected result states: %+v", result)
	}
	action, fault, inverter := fixture.states(t)
	if action != "executing" || fault != "mitigated" || inverter != "isolated" {
		t.Fatalf("persisted states mismatch: %s %s %s", action, fault, inverter)
	}
	if result.Action.Version != fixture.action.Version+1 {
		t.Fatalf("expected action version bump, got %d", result.Action.Version)
	}
	if total := fixture.auditCount(t); total != 3 {
		t.Fatalf("expected 3 audit rows, got %d", total)
	}
	var actions []string
	if err := fixture.db.Model(&model.AuditLog{}).Where("action = ?", "confirm").Pluck("entity_type", &actions).Error; err != nil {
		t.Fatalf("load audit entity types: %v", err)
	}
	if len(actions) != 3 {
		t.Fatalf("expected confirm audits for all three entities, got %v", actions)
	}
}

func TestConfirmInterlockSucceedsOnlyOnce(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "tripped")
	if _, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-1"); err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	// A repeated confirm with the stale version fails on the optimistic lock.
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-2")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected repeated confirm to be rejected, got %v", err)
	}
	action, fault, inverter := fixture.states(t)
	if action != "executing" || fault != "mitigated" || inverter != "isolated" {
		t.Fatalf("states changed after repeated confirm: %s %s %s", action, fault, inverter)
	}
	if total := fixture.auditCount(t); total != 3 {
		t.Fatalf("expected audits to stay at 3, got %d", total)
	}
}

func TestConfirmInterlockVersionConflictKeepsEverything(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	input := fixture.confirmInput()
	input.ExpectedVersion = fixture.action.Version + 9
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, input, "operator", "req-1")
	if !errors.Is(err, repository.ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	action, fault, inverter := fixture.states(t)
	if action != "confirmed" || fault != "acknowledged" || inverter != "warning" {
		t.Fatalf("states must stay untouched on conflict: %s %s %s", action, fault, inverter)
	}
	if total := fixture.auditCount(t); total != 0 {
		t.Fatalf("expected no audit rows on conflict, got %d", total)
	}
}

func TestConfirmInterlockRequiresAcknowledgedFault(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "open", "warning")
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-1")
	if !errors.Is(err, ErrInterlockPrecondition) {
		t.Fatalf("expected interlock precondition error, got %v", err)
	}
	action, fault, inverter := fixture.states(t)
	if action != "confirmed" || fault != "open" || inverter != "warning" {
		t.Fatalf("states must stay untouched: %s %s %s", action, fault, inverter)
	}
	if total := fixture.auditCount(t); total != 0 {
		t.Fatalf("expected no audit rows, got %d", total)
	}
}

func TestConfirmInterlockRequiresWarningOrTrippedInverter(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "online")
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-1")
	if !errors.Is(err, ErrInterlockPrecondition) {
		t.Fatalf("expected interlock precondition error, got %v", err)
	}
	action, fault, inverter := fixture.states(t)
	if action != "confirmed" || fault != "acknowledged" || inverter != "online" {
		t.Fatalf("states must stay untouched: %s %s %s", action, fault, inverter)
	}
	if total := fixture.auditCount(t); total != 0 {
		t.Fatalf("expected no audit rows, got %d", total)
	}
}

func TestConfirmInterlockRejectsIsolatedInverter(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "isolated")
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-1")
	if !errors.Is(err, ErrInterlockPrecondition) {
		t.Fatalf("expected interlock precondition error, got %v", err)
	}
}

func TestConfirmInterlockMissingLinkedFault(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	if err := fixture.db.Delete(&model.FaultEvent{}, fixture.fault.ID).Error; err != nil {
		t.Fatalf("delete fault: %v", err)
	}
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-1")
	if !errors.Is(err, ErrInterlockLink) {
		t.Fatalf("expected interlock link error, got %v", err)
	}
	var action model.MitigationAction
	if err := fixture.db.First(&action, fixture.action.ID).Error; err != nil {
		t.Fatalf("reload action: %v", err)
	}
	if action.Status != "confirmed" {
		t.Fatalf("action must stay untouched, got %s", action.Status)
	}
	if total := fixture.auditCount(t); total != 0 {
		t.Fatalf("expected no audit rows, got %d", total)
	}
}

func TestConfirmInterlockRequiresSecondConfirmation(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	input := fixture.confirmInput()
	input.Confirmed = false
	_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, input, "operator", "req-1")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if total := fixture.auditCount(t); total != 0 {
		t.Fatalf("expected no audit rows, got %d", total)
	}
}

func TestTransitionIntoExecutingRequiresInterlock(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	input := dto.TransitionRequest{Status: "executing", ExpectedVersion: fixture.action.Version, Reason: "绕过联锁的直接迁移", Confirmed: true}
	_, err := fixture.service.Transition(context.Background(), fixture.action.ID, input, "operator", "req-1")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected direct executing transition to be rejected, got %v", err)
	}
	action, fault, inverter := fixture.states(t)
	if action != "confirmed" || fault != "acknowledged" || inverter != "warning" {
		t.Fatalf("states must stay untouched: %s %s %s", action, fault, inverter)
	}
}

func TestConfirmInterlockConcurrentConfirmSucceedsOnce(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	const attempts = 8
	results := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			_, err := fixture.service.ConfirmInterlock(context.Background(), fixture.action.ID, fixture.confirmInput(), "operator", "req-concurrent")
			results <- err
		}()
	}
	succeeded := 0
	for i := 0; i < attempts; i++ {
		if err := <-results; err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("expected exactly one successful confirm, got %d", succeeded)
	}
	action, fault, inverter := fixture.states(t)
	if action != "executing" || fault != "mitigated" || inverter != "isolated" {
		t.Fatalf("final states mismatch: %s %s %s", action, fault, inverter)
	}
	if total := fixture.auditCount(t); total != 3 {
		t.Fatalf("expected exactly 3 audit rows, got %d", total)
	}
}
