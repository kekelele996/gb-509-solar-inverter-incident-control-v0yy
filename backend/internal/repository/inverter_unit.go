package repository

import (
	"context"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"gorm.io/gorm"
)

// InverterUnitRepository owns all persistence operations for 逆变器.
type InverterUnitRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.InverterUnit], error)
	Get(context.Context, uint) (model.InverterUnit, error)
	Create(context.Context, *model.InverterUnit) error
	Update(context.Context, uint, uint, *model.InverterUnit) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type inverterUnitRepository struct {
	store *Store[model.InverterUnit]
}

func NewInverterUnitRepository(db *gorm.DB) InverterUnitRepository {
	return &inverterUnitRepository{store: NewStore[model.InverterUnit](db)}
}

func (r *inverterUnitRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.InverterUnit], error) {
	return r.store.List(ctx, q)
}
func (r *inverterUnitRepository) Get(ctx context.Context, id uint) (model.InverterUnit, error) {
	return r.store.Get(ctx, id)
}
func (r *inverterUnitRepository) Create(ctx context.Context, item *model.InverterUnit) error {
	return r.store.Create(ctx, item)
}
func (r *inverterUnitRepository) Update(ctx context.Context, id, version uint, item *model.InverterUnit) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *inverterUnitRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *inverterUnitRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
