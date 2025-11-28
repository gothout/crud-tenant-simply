package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	CreateAcessToken(ctx context.Context, m AcessToken) error
	RevokeAcessToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	GetAcessToken(ctx context.Context, token string) (AcessToken, error)
}

type repositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{
		db: db,
	}
}

func (r *repositoryImpl) CreateAcessToken(ctx context.Context, m AcessToken) error {
	query := r.db.WithContext(ctx).Create(&m)

	if query.Error == nil {
		return nil
	}
	var pgErr *pgconn.PgError

	if errors.As(query.Error, &pgErr) {
		if pgErr.Code == "23505" {
			return ErrTokenDuplicated
		}
		if pgErr.Code == "23503" {
			return ErrTokenDuplicated
		}
		return pgErr
	}
	return query.Error
}

func (r *repositoryImpl) RevokeAcessToken(ctx context.Context, token string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&AcessToken{}).
		Where("token = ?", token).
		Update("expire_date", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no rows affected")
	}
	return nil
}

func (r *repositoryImpl) RevokeAllUserTokens(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&AcessToken{}).
		Where("user_uuid = ? AND expire_date > ?", userID, now).
		Update("expire_date", now)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *repositoryImpl) GetAcessToken(ctx context.Context, token string) (AcessToken, error) {
	var m AcessToken
	result := r.db.WithContext(ctx).First(&m, "token = ?", token)
	if result.Error != nil {
		return AcessToken{}, result.Error
	}
	return m, nil
}
