package user

import (
	"context"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/pkg/util"
	"time"
)

type Service interface {
	Create(ctx context.Context, user User) (User, error)
}

type serviceImpl struct {
	Repository Repository
}

func NewService(repository Repository) Service {
	return &serviceImpl{
		Repository: repository,
	}
}

func (s *serviceImpl) Create(ctx context.Context, user User) (User, error) {
	t, err := tenant.MustUse().Service.Read(ctx, user.Tenant)
	if err != nil {
		return User{}, err
	}
	hashPwd, err := util.UsePassword().Hash(user.Password)
	if err != nil {
		return User{}, err
	}
	newUser := User{
		TenantUUID: &t.UUID,
		Name:       user.Name,
		Email:      user.Email,
		Password:   hashPwd,
		Role:       user.Role,
		Live:       user.Live,
		CreateAt:   time.Now().UTC(),
		UpdateAt:   time.Now().UTC(),
		Tenant:     t,
	}

	return s.Repository.Create(ctx, newUser)
}
