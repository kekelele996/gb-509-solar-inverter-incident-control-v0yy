package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/config"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Config, log *slog.Logger) (*gorm.DB, *redis.Client, error) {
	var dialector gorm.Dialector
	switch cfg.DatabaseDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseDSN)
	case "mysql":
		dialector = mysql.Open(cfg.DatabaseDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DatabaseDSN)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
	logLevel := logger.Warn
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 20; attempt++ {
		db, err = gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logLevel)})
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil && sqlDB.PingContext(ctx) == nil {
				break
			}
			if dbErr != nil {
				err = dbErr
			} else {
				err = sqlDB.PingContext(ctx)
			}
		}
		log.Warn("database not ready", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, nil, err
	}
	if err := Seed(ctx, db); err != nil {
		return nil, nil, err
	}
	var redisClient *redis.Client
	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, nil, fmt.Errorf("connect redis: %w", err)
		}
	}
	return db, redisClient, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{}, &model.AuditLog{},
		&model.SolarSite{},
		&model.InverterUnit{},
		&model.FaultEvent{},
		&model.MitigationAction{},
	)
}

func Seed(ctx context.Context, db *gorm.DB) error {
	var users int64
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&users).Error; err != nil {
		return err
	}
	if users == 0 {
		password, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		seedUsers := []model.User{
			{Username: "admin", DisplayName: "系统管理员", PasswordHash: string(password), Role: model.RoleAdmin, Active: true},
			{Username: "reviewer", DisplayName: "质量复核员", PasswordHash: string(password), Role: model.RoleReviewer, Active: true},
			{Username: "operator", DisplayName: "现场操作员", PasswordHash: string(password), Role: model.RoleOperator, Active: true},
			{Username: "viewer", DisplayName: "只读观察员", PasswordHash: string(password), Role: model.RoleViewer, Active: true},
		}
		if err := db.WithContext(ctx).Create(&seedUsers).Error; err != nil {
			return err
		}
	}

	if err := seedSolarSite(ctx, db); err != nil {
		return err
	}

	if err := seedInverterUnit(ctx, db); err != nil {
		return err
	}

	if err := seedFaultEvent(ctx, db); err != nil {
		return err
	}

	if err := seedMitigationAction(ctx, db); err != nil {
		return err
	}

	return nil
}

func seedSolarSite(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.SolarSite{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.SolarSite{

		{BaseModel: model.BaseModel{Code: "SS-001", Name: "光伏场站示例一", Status: "online", Version: 1,
			Description: "用于启动验证和主要流程演示的光伏场站记录"}, Facility: "光伏逆变器故障处置控制区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-01"},

		{BaseModel: model.BaseModel{Code: "SS-002", Name: "光伏场站示例二", Status: "limited", Version: 1,
			Description: "用于启动验证和主要流程演示的光伏场站记录"}, Facility: "光伏逆变器故障处置控制区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-02"},

		{BaseModel: model.BaseModel{Code: "SS-003", Name: "光伏场站示例三", Status: "offline", Version: 1,
			Description: "用于启动验证和主要流程演示的光伏场站记录"}, Facility: "光伏逆变器故障处置控制区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedInverterUnit(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.InverterUnit{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.InverterUnit{

		{BaseModel: model.BaseModel{Code: "IU-001", Name: "逆变器示例一", Status: "online", Version: 1,
			Description: "用于启动验证和主要流程演示的逆变器记录"}, Facility: "光伏逆变器故障处置控制区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-01"},

		{BaseModel: model.BaseModel{Code: "IU-002", Name: "逆变器示例二", Status: "warning", Version: 1,
			Description: "用于启动验证和主要流程演示的逆变器记录"}, Facility: "光伏逆变器故障处置控制区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-02"},

		{BaseModel: model.BaseModel{Code: "IU-003", Name: "逆变器示例三", Status: "tripped", Version: 1,
			Description: "用于启动验证和主要流程演示的逆变器记录"}, Facility: "光伏逆变器故障处置控制区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedFaultEvent(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.FaultEvent{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.FaultEvent{

		{BaseModel: model.BaseModel{Code: "FE-001", Name: "故障事件示例一", Status: "open", Version: 1,
			Description: "用于启动验证和主要流程演示的故障事件记录"}, Facility: "光伏逆变器故障处置控制区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-01"},

		{BaseModel: model.BaseModel{Code: "FE-002", Name: "故障事件示例二", Status: "acknowledged", Version: 1,
			Description: "用于启动验证和主要流程演示的故障事件记录"}, Facility: "光伏逆变器故障处置控制区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-02"},

		{BaseModel: model.BaseModel{Code: "FE-003", Name: "故障事件示例三", Status: "mitigated", Version: 1,
			Description: "用于启动验证和主要流程演示的故障事件记录"}, Facility: "光伏逆变器故障处置控制区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedMitigationAction(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.MitigationAction{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.MitigationAction{

		{BaseModel: model.BaseModel{Code: "MA-001", Name: "处置动作示例一", Status: "draft", Version: 1,
			Description: "用于启动验证和主要流程演示的处置动作记录"}, Facility: "光伏逆变器故障处置控制区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-01"},

		{BaseModel: model.BaseModel{Code: "MA-002", Name: "处置动作示例二", Status: "confirmed", Version: 1,
			Description: "用于启动验证和主要流程演示的处置动作记录"}, Facility: "光伏逆变器故障处置控制区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-02"},

		{BaseModel: model.BaseModel{Code: "MA-003", Name: "处置动作示例三", Status: "executing", Version: 1,
			Description: "用于启动验证和主要流程演示的处置动作记录"}, Facility: "光伏逆变器故障处置控制区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-509-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}
