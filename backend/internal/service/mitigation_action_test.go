package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/config"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/constants"
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

func TestRemoteActionRequiresSecondConfirmation(t *testing.T) {
	input := dto.TransitionRequest{Status: "confirmed", ExpectedVersion: 1, Reason: "operator approval"}
	err := requireSecondConfirmation(input)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected missing confirmation to fail, got %v", err)
	}
	input.Confirmed = true
	if err := requireSecondConfirmation(input); err != nil {
		t.Fatalf("expected explicit confirmation to pass, got %v", err)
	}
}

func TestInterlockConfirmationAtomicallyUpdatesAllObjects(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	ctx := context.Background()

	result, err := fixture.service.Transition(ctx, fixture.action.ID, dto.TransitionRequest{
		Status: "executing", ExpectedVersion: 1, Reason: "确认故障联锁", Confirmed: true,
	}, "operator", "request-success")
	if err != nil {
		t.Fatalf("expected interlock confirmation to succeed: %v", err)
	}
	if result.Action.Status != "executing" || result.Fault == nil || result.Fault.Status != "mitigated" ||
		result.Inverter == nil || result.Inverter.Status != "isolated" {
		t.Fatalf("unexpected interlock result: %+v", result)
	}

	action := mustGetAction(t, fixture, fixture.action.ID)
	fault := mustGetFault(t, fixture, fixture.fault.ID)
	inverter := mustGetInverter(t, fixture, fixture.inverter.ID)
	if action.Status != "executing" || action.Version != 2 {
		t.Fatalf("action not committed: status=%s version=%d", action.Status, action.Version)
	}
	if fault.Status != "mitigated" || fault.Version != 2 {
		t.Fatalf("fault not committed: status=%s version=%d", fault.Status, fault.Version)
	}
	if inverter.Status != "isolated" || inverter.Version != 2 {
		t.Fatalf("inverter not committed: status=%s version=%d", inverter.Status, inverter.Version)
	}
	assertAuditCount(t, fixture.db, 3)
}

func TestInterlockConfirmationIsIdempotent(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	ctx := context.Background()
	input := dto.TransitionRequest{Status: "executing", ExpectedVersion: 1, Reason: "确认故障联锁", Confirmed: true}
	if _, err := fixture.service.Transition(ctx, fixture.action.ID, input, "operator", "request-1"); err != nil {
		t.Fatalf("first confirmation failed: %v", err)
	}

	_, err := fixture.service.Transition(ctx, fixture.action.ID, input, "operator", "request-2")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected repeated confirmation to be rejected, got %v", err)
	}
	assertInterlockReadIsCommitted(t, fixture)
	assertAuditCount(t, fixture.db, 3)
}

func TestInterlockRejectsStaleActionVersion(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	ctx := context.Background()

	if err := fixture.db.Model(&model.MitigationAction{}).Where("id = ?", fixture.action.ID).
		Updates(map[string]any{"version": 2, "status": "confirmed"}).Error; err != nil {
		t.Fatalf("change action version: %v", err)
	}
	_, err := fixture.service.Transition(ctx, fixture.action.ID, dto.TransitionRequest{
		Status: "executing", ExpectedVersion: 1, Reason: "旧版本确认", Confirmed: true,
	}, "operator", "request-stale")
	if !errors.Is(err, repository.ErrVersionConflict) {
		t.Fatalf("expected version conflict, got %v", err)
	}
	if fault := mustGetFault(t, fixture, fixture.fault.ID); fault.Status != "acknowledged" || fault.Version != 1 {
		t.Fatalf("fault changed after stale request: %+v", fault)
	}
	assertAuditCount(t, fixture.db, 0)
}

func TestInterlockRejectsUnacknowledgedFault(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "open", "warning")
	_, err := fixture.service.Transition(context.Background(), fixture.action.ID, dto.TransitionRequest{
		Status: "executing", ExpectedVersion: 1, Reason: "故障未确认", Confirmed: true,
	}, "operator", "request-fault")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected unacknowledged fault rejection, got %v", err)
	}
	assertInterlockReadIsUnchanged(t, fixture)
}

func TestInterlockRejectsInverterNotWarningOrTripped(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "online")
	_, err := fixture.service.Transition(context.Background(), fixture.action.ID, dto.TransitionRequest{
		Status: "executing", ExpectedVersion: 1, Reason: "逆变器在线", Confirmed: true,
	}, "operator", "request-inverter")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid inverter state rejection, got %v", err)
	}
	assertInterlockReadIsUnchanged(t, fixture)
}

func TestInterlockRejectsMissingRelatedFault(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	action := fixture.action
	action.Facility = "其他场站"
	if err := fixture.db.Model(&model.MitigationAction{}).Where("id = ?", action.ID).Update("facility", action.Facility).Error; err != nil {
		t.Fatalf("move action facility: %v", err)
	}
	_, err := fixture.service.Transition(context.Background(), action.ID, dto.TransitionRequest{
		Status: "executing", ExpectedVersion: 1, Reason: "场站不匹配", Confirmed: true,
	}, "operator", "request-missing")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected missing related fault rejection, got %v", err)
	}
	assertInterlockReadIsUnchanged(t, fixture)
}

func TestAuditWriteFailureRollsBackEntireInterlock(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	if err := fixture.db.Exec(`CREATE TRIGGER fail_audit_insert BEFORE INSERT ON audit_logs
		BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END`).Error; err != nil {
		t.Fatalf("create audit failure trigger: %v", err)
	}
	_, err := fixture.service.Transition(context.Background(), fixture.action.ID, dto.TransitionRequest{
		Status: "executing", ExpectedVersion: 1, Reason: "审计失败", Confirmed: true,
	}, "operator", "request-audit-failure")
	if err == nil {
		t.Fatal("expected audit failure to abort the transaction")
	}
	assertInterlockReadIsUnchanged(t, fixture)
	assertAuditCount(t, fixture.db, 0)
}

func TestConcurrentInterlockConfirmationSucceedsOnce(t *testing.T) {
	fixture := newInterlockFixture(t, "confirmed", "acknowledged", "warning")
	ctx := context.Background()
	const requests = 10
	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan error, requests)
	for range requests {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := fixture.service.Transition(ctx, fixture.action.ID, dto.TransitionRequest{
				Status: "executing", ExpectedVersion: 1, Reason: "并发确认", Confirmed: true,
			}, "operator", "request-concurrent")
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if !errors.Is(err, repository.ErrVersionConflict) && !errors.Is(err, ErrInvalidTransition) {
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one successful confirmation, got %d", successes)
	}
	assertInterlockReadIsCommitted(t, fixture)
	assertAuditCount(t, fixture.db, 3)
}

func newInterlockFixture(t *testing.T, actionStatus, faultStatus, inverterStatus string) interlockFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "interlock.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	// Serialize SQLite writers so an optimistic-lock conflict does not surface as
	// SQLITE_BUSY before the application can apply the version predicate.
	sqlDB.SetMaxOpenConns(1)
	for _, pragma := range []string{"PRAGMA journal_mode=WAL", "PRAGMA busy_timeout=5000"} {
		if err := db.Exec(pragma).Error; err != nil {
			t.Fatalf("%s: %v", pragma, err)
		}
	}
	if err := db.AutoMigrate(&model.AuditLog{}, &model.InverterUnit{}, &model.FaultEvent{}, &model.MitigationAction{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	now := time.Now().UTC()
	fault := model.FaultEvent{
		BaseModel: model.BaseModel{Code: "FE-LOCK-1", Name: "联锁故障", Status: faultStatus, Version: 1, CreatedAt: now, UpdatedAt: now},
		Facility:  "一号光伏场站", Owner: "运行一组", Category: "过温", RiskLevel: "high",
		MetricValue: 78, MetricUnit: "℃", EffectiveAt: now, Evidence: "故障已确认", RelatedCode: "REL-LOCK-1",
	}
	inverter := model.InverterUnit{
		BaseModel: model.BaseModel{Code: "INV-LOCK-1", Name: "联锁逆变器", Status: inverterStatus, Version: 1, CreatedAt: now, UpdatedAt: now},
		Facility:  "一号光伏场站", Owner: "运行一组", Category: "逆变器", RiskLevel: "high",
		MetricValue: 78, MetricUnit: "℃", EffectiveAt: now, Evidence: "设备异常", RelatedCode: "REL-LOCK-1",
	}
	action := model.MitigationAction{
		BaseModel: model.BaseModel{Code: "ACT-LOCK-1", Name: "联锁处置动作", Status: actionStatus, Version: 1, CreatedAt: now, UpdatedAt: now},
		Facility:  "一号光伏场站", Owner: "运行一组", Category: "远程隔离", RiskLevel: "high",
		MetricValue: 1, MetricUnit: "台", EffectiveAt: now, Evidence: "处置方案已复核", RelatedCode: "REL-LOCK-1",
	}
	if err := db.Create(&fault).Error; err != nil {
		t.Fatalf("create fault: %v", err)
	}
	if err := db.Create(&inverter).Error; err != nil {
		t.Fatalf("create inverter: %v", err)
	}
	if err := db.Create(&action).Error; err != nil {
		t.Fatalf("create action: %v", err)
	}

	security := NewSecurityService(repository.NewSecurityRepository(db), config.Config{})
	svc := NewMitigationActionService(
		repository.NewMitigationActionRepository(db),
		repository.NewFaultEventRepository(db),
		repository.NewInverterUnitRepository(db),
		repository.NewTransactionManager(db),
		security,
	)
	return interlockFixture{db: db, service: svc, action: action, fault: fault, inverter: inverter}
}

func mustGetAction(t *testing.T, fixture interlockFixture, id uint) model.MitigationAction {
	t.Helper()
	var item model.MitigationAction
	if err := fixture.db.First(&item, id).Error; err != nil {
		t.Fatalf("read action: %v", err)
	}
	return item
}

func mustGetFault(t *testing.T, fixture interlockFixture, id uint) model.FaultEvent {
	t.Helper()
	var item model.FaultEvent
	if err := fixture.db.First(&item, id).Error; err != nil {
		t.Fatalf("read fault: %v", err)
	}
	return item
}

func mustGetInverter(t *testing.T, fixture interlockFixture, id uint) model.InverterUnit {
	t.Helper()
	var item model.InverterUnit
	if err := fixture.db.First(&item, id).Error; err != nil {
		t.Fatalf("read inverter: %v", err)
	}
	return item
}

func assertInterlockReadIsUnchanged(t *testing.T, fixture interlockFixture) {
	t.Helper()
	action := mustGetAction(t, fixture, fixture.action.ID)
	fault := mustGetFault(t, fixture, fixture.fault.ID)
	inverter := mustGetInverter(t, fixture, fixture.inverter.ID)
	if action.Status != fixture.action.Status || action.Version != 1 ||
		fault.Status != fixture.fault.Status || fault.Version != 1 ||
		inverter.Status != fixture.inverter.Status || inverter.Version != 1 {
		t.Fatalf("objects changed after rejected request: action=%s/%d fault=%s/%d inverter=%s/%d",
			action.Status, action.Version, fault.Status, fault.Version, inverter.Status, inverter.Version)
	}
	assertAuditCount(t, fixture.db, 0)
}

func assertInterlockReadIsCommitted(t *testing.T, fixture interlockFixture) {
	t.Helper()
	action := mustGetAction(t, fixture, fixture.action.ID)
	fault := mustGetFault(t, fixture, fixture.fault.ID)
	inverter := mustGetInverter(t, fixture, fixture.inverter.ID)
	if action.Status != "executing" || action.Version != 2 ||
		fault.Status != string(constants.FaultStateMitigated) || fault.Version != 2 ||
		inverter.Status != string(constants.InverterStateIsolated) || inverter.Version != 2 {
		t.Fatalf("committed interlock mismatch: action=%s/%d fault=%s/%d inverter=%s/%d",
			action.Status, action.Version, fault.Status, fault.Version, inverter.Status, inverter.Version)
	}
}

func assertAuditCount(t *testing.T, db *gorm.DB, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.AuditLog{}).Count(&count).Error; err != nil {
		t.Fatalf("count audits: %v", err)
	}
	if count != expected {
		t.Fatalf("expected %d audit rows, got %d", expected, count)
	}
}
