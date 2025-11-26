package tenant

import "context"

type Service interface {
	Create(ctx context.Context, tenant Tenant) (Tenant, error)
}

type implService struct {
	Repository Repository
}

func NewService(repository Repository) Service {
	return &implService{
		Repository: repository,
	}
}

func (s *implService) Create(ctx context.Context, tenant Tenant) (Tenant, error) {
	return s.Repository.Create(ctx, tenant)
}
