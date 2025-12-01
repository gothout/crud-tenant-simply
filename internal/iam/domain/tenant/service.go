package tenant

import (
	"context"
	"tenant-crud-simply/internal/iam/domain/model"
)

type Service interface {
	Create(ctx context.Context, tenant model.Tenant) (model.Tenant, error)
	Read(ctx context.Context, tenant model.Tenant) (model.Tenant, error)
	List(ctx context.Context, page, pageSize int) ([]model.Tenant, error)
	Update(ctx context.Context, m *model.Tenant) (model.Tenant, error)
	Delete(ctx context.Context, m model.Tenant) error
}

type implService struct {
	Repository Repository
}

func NewService(repository Repository) Service {
	return &implService{
		Repository: repository,
	}
}

func (s *implService) Create(ctx context.Context, tenant model.Tenant) (model.Tenant, error) {
	return s.Repository.Create(ctx, tenant)
}

func (s *implService) Read(ctx context.Context, tenant model.Tenant) (model.Tenant, error) {
	return s.Repository.Read(ctx, tenant)
}

func (s *implService) List(ctx context.Context, page, pageSize int) ([]model.Tenant, error) {
	return s.Repository.List(ctx, page, pageSize)
}

func (s *implService) Update(ctx context.Context, m *model.Tenant) (model.Tenant, error) {
	return s.Repository.Update(ctx, m)
}
func (s *implService) Delete(ctx context.Context, m model.Tenant) error {
	return s.Repository.Delete(ctx, m)
}
