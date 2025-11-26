package user

import (
	"context"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/pkg/util"
	"time"
)

type Service interface {
	Create(ctx context.Context, user User) (User, error)
	Read(ctx context.Context, user User) (User, error)
	List(ctx context.Context, page, pageSize int) ([]User, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, user User) error
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

func (s *serviceImpl) Read(ctx context.Context, user User) (User, error) {
	return s.Repository.Read(ctx, user)
}

func (s *serviceImpl) List(ctx context.Context, page, pageSize int) ([]User, error) {
	return s.Repository.List(ctx, page, pageSize)
}

func (s *serviceImpl) Update(ctx context.Context, user User) (User, error) {
	if user.Password != "" {
		hashPwd, err := util.UsePassword().Hash(user.Password)
		if err != nil {
			return User{}, err
		}
		user.Password = hashPwd
	}

	user.UpdateAt = time.Now().UTC()

	return s.Repository.Update(ctx, user)
}

func (s *serviceImpl) Delete(ctx context.Context, user User) error {
	return s.Repository.Delete(ctx, user)
}
