package tenant

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, tenant Tenant) (Tenant, error)
}

type implRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &implRepository{db: db}
}

func (r *implRepository) Create(ctx context.Context, m Tenant) (Tenant, error) {
	result := r.db.WithContext(ctx).Create(&m)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			return Tenant{}, ErrDocumentDuplicated
		}
		return Tenant{}, result.Error
	}

	if result.RowsAffected == 0 {
		return Tenant{}, fmt.Errorf("no rows affected")
	}
	return m, nil
}
