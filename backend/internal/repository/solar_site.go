package repository

import (
	"context"

	"github.com/blueship581/solar-inverter-incident-control/backend/internal/dto"
	"github.com/blueship581/solar-inverter-incident-control/backend/internal/model"
	"gorm.io/gorm"
)

// SolarSiteRepository owns all persistence operations for 光伏场站.
type SolarSiteRepository interface {
	List(context.Context, dto.PageQuery) (Page[model.SolarSite], error)
	Get(context.Context, uint) (model.SolarSite, error)
	Create(context.Context, *model.SolarSite) error
	Update(context.Context, uint, uint, *model.SolarSite) error
	Delete(context.Context, uint) error
	CountByStatus(context.Context) (map[string]int64, error)
}

type solarSiteRepository struct {
	store *Store[model.SolarSite]
}

func NewSolarSiteRepository(db *gorm.DB) SolarSiteRepository {
	return &solarSiteRepository{store: NewStore[model.SolarSite](db)}
}

func (r *solarSiteRepository) List(ctx context.Context, q dto.PageQuery) (Page[model.SolarSite], error) {
	return r.store.List(ctx, q)
}
func (r *solarSiteRepository) Get(ctx context.Context, id uint) (model.SolarSite, error) {
	return r.store.Get(ctx, id)
}
func (r *solarSiteRepository) Create(ctx context.Context, item *model.SolarSite) error {
	return r.store.Create(ctx, item)
}
func (r *solarSiteRepository) Update(ctx context.Context, id, version uint, item *model.SolarSite) error {
	return r.store.Update(ctx, id, version, item)
}
func (r *solarSiteRepository) Delete(ctx context.Context, id uint) error {
	return r.store.Delete(ctx, id)
}
func (r *solarSiteRepository) CountByStatus(ctx context.Context) (map[string]int64, error) {
	return r.store.CountByStatus(ctx)
}
