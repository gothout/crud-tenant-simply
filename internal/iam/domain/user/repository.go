package user

import (
	"context"
	"errors"
	"tenant-crud-simply/internal/iam/domain/tenant"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user User) (User, error)
}

type repositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{
		db: db,
	}
}

func (r *repositoryImpl) Create(ctx context.Context, user User) (User, error) {
	result := r.db.WithContext(ctx).Create(&user)
	if result.Error == nil {
		return user, nil
	}
	var pgErr *pgconn.PgError

	if errors.As(result.Error, &pgErr) {
		if pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_email_key":
				return User{}, ErrEmailDuplicated
			default:
				return User{}, result.Error
			}
		}
		if pgErr.Code == "23503" {
			return User{}, tenant.ErrNotFound
		}
	}
	return User{}, result.Error

}
