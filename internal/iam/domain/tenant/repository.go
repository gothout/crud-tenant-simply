package tenant

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, tenant Tenant) (Tenant, error)
	Read(ctx context.Context, m Tenant) (Tenant, error)
	List(ctx context.Context, page, pageSize int) ([]Tenant, error)
	Update(ctx context.Context, m *Tenant) (Tenant, error)
	Delete(ctx context.Context, m Tenant) error
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

func (r *implRepository) Read(ctx context.Context, m Tenant) (Tenant, error) {
	query := r.db.WithContext(ctx).Model(&Tenant{})
	if m.UUID != uuid.Nil {
		query = query.First(&m, "uuid = ?", m.UUID)
	} else if m.Document != "" {
		query = query.Where("document = ?", m.Document).First(&m)
	} else {
		return Tenant{}, ErrInvalidInput
	}
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return Tenant{}, ErrNotFound
		}
		return Tenant{}, fmt.Errorf("erro ao ler tenant: %w", query.Error)
	}
	return m, nil
}

func (r *implRepository) List(ctx context.Context, page, pageSize int) ([]Tenant, error) {
	var listTenant []Tenant
	query := r.db.WithContext(ctx).Model(&Tenant{})
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	result := query.Limit(pageSize).Offset(offset).Find(&listTenant)
	if result.Error != nil {
		return nil, result.Error
	}
	return listTenant, nil
}

func (r *implRepository) Update(ctx context.Context, m *Tenant) (Tenant, error) {
	if m.UUID == uuid.Nil {
		return Tenant{}, ErrInvalidInput
	}

	updateModel := Tenant{
		Name:     m.Name,
		Document: m.Document,
		Live:     m.Live,
		UpdateAt: time.Now().UTC(),
	}
	result := r.db.WithContext(ctx).
		Where("uuid = ?", m.UUID).
		Select("Name", "Document", "Live", "UpdateAt").
		Updates(updateModel)

	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value violates unique constraint") {
			return Tenant{}, ErrDocumentDuplicated
		}
		return Tenant{}, result.Error
	}

	if result.RowsAffected == 0 {
		existingTenant, err := r.Read(ctx, Tenant{UUID: m.UUID})
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return Tenant{}, ErrNotFound
			}
			return Tenant{}, fmt.Errorf("falha ao verificar se o tenant existe: %w", err)
		}

		return existingTenant, nil
	}
	updatedTenant, err := r.Read(ctx, Tenant{UUID: m.UUID})
	if err != nil {
		return updatedTenant, fmt.Errorf("falha ao ler tenant atualizado: %w", err)
	}

	return updatedTenant, nil
}

func (r *implRepository) Delete(ctx context.Context, m Tenant) error {
	if m.UUID == uuid.Nil && m.Document == "" {
		return ErrInvalidInput
	}
	query := r.db.WithContext(ctx)
	if m.UUID != uuid.Nil {
		query = query.Where("uuid = ?", m.UUID)
	} else {
		query = query.Where("document = ?", m.Document)
	}
	result := query.Delete(&Tenant{})
	if result.Error != nil {
		return fmt.Errorf("falha ao deletar tenant: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
