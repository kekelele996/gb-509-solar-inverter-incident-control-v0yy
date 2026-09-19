package repository

import (
	"context"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"gorm.io/gorm"
)

// MitigationActionRepository owns all persistence operations for 处置动作.
type MitigationActionRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.MitigationAction], error)
	Get(context.Context, uint) (model.MitigationAction, error)
	Create(context.Context, *model.MitigationAction) error
	Update(context.Context, uint, uint, *model.MitigationAction) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type mitigationActionRepository struct {
	store *Store[model.MitigationAction]
}

func NewMitigationActionRepository(db *gorm.DB) MitigationActionRepository {
	return &mitigationActionRepository{store: NewStore[model.MitigationAction](db)}
}

func (r *mitigationActionRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.MitigationAction], error) {
	return r.store.List(ctx, q)
}
func (r *mitigationActionRepository) Get(ctx context.Context, id uint) (model.MitigationAction, error) {
	return r.store.Get(ctx, id)
}
func (r *mitigationActionRepository) Create(ctx context.Context, item *model.MitigationAction) error {
	return r.store.Create(ctx, item)
}
func (r *mitigationActionRepository) Update(ctx context.Context, id, version uint, item *model.MitigationAction) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *mitigationActionRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *mitigationActionRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
