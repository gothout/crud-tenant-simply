package auditoria_log

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Save(ctx context.Context, entry AuditLog) error
}

type repositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Save(ctx context.Context, entry AuditLog) error {
	return r.db.WithContext(ctx).Create(&entry).Error
}
