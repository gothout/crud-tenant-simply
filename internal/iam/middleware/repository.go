package middleware

import (
	"context"
	"database/sql"
	"errors"
	"tenant-crud-simply/internal/iam/domain/tenant"
	"tenant-crud-simply/internal/iam/domain/user"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	GetLogin(ctx context.Context, token string) (*Login, error)
}

type repositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

type loginQueryResult struct {
	Token            string         `gorm:"column:token"`
	Expiry           time.Time      `gorm:"column:expire_date"`
	UserUUID         uuid.UUID      `gorm:"column:user_uuid"`
	UserTenantUUID   *uuid.UUID     `gorm:"column:tenant_uuid"`
	UserName         string         `gorm:"column:user_name"`
	UserEmail        string         `gorm:"column:user_email"`
	UserPasswordHash string         `gorm:"column:password_hash"`
	UserRole         user.UserRole  `gorm:"column:role"`
	UserLive         bool           `gorm:"column:live"`
	UserCreateAt     time.Time      `gorm:"column:create_at"`
	UserUpdateAt     time.Time      `gorm:"column:update_at"`
	TenantName       sql.NullString `gorm:"column:tenant_name"`
	TenantDocument   sql.NullString `gorm:"column:tenant_document"`
	TenantLive       sql.NullBool   `gorm:"column:tenant_live"`
	TenantCreateAt   sql.NullTime   `gorm:"column:tenant_create_at"`
	TenantUpdateAt   sql.NullTime   `gorm:"column:tenant_update_at"`
}

const loginQuery = `
SELECT
        at.token,
        at.expire_date,
        at.user_uuid,
        u.tenant_uuid,
        u.name AS user_name,
        u.email AS user_email,
        u.password_hash,
        u.role,
        u.live,
        u.create_at,
        u.update_at,
        t.name AS tenant_name,
        t.document AS tenant_document,
        t.live AS tenant_live,
        t.create_at AS tenant_create_at,
        t.update_at AS tenant_update_at
FROM users_acess_tokens AS at
INNER JOIN users AS u ON u.uuid = at.user_uuid
LEFT JOIN tenant AS t ON t.uuid = u.tenant_uuid
WHERE at.token = ?
LIMIT 1`

func (r *repositoryImpl) GetLogin(ctx context.Context, token string) (*Login, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	var result loginQueryResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Raw(loginQuery, token).Scan(&result)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	login := &Login{
		User: user.User{
			UUID:       result.UserUUID,
			TenantUUID: result.UserTenantUUID,
			Name:       result.UserName,
			Email:      result.UserEmail,
			Password:   result.UserPasswordHash,
			Role:       result.UserRole,
			Live:       result.UserLive,
			CreateAt:   result.UserCreateAt,
			UpdateAt:   result.UserUpdateAt,
		},
		AcessToken: AcessToken{
			UserUUID: &result.UserUUID,
			Token:    result.Token,
			Expiry:   result.Expiry,
		},
	}

	if result.UserTenantUUID != nil {
		tenant := tenant.Tenant{
			UUID:     *result.UserTenantUUID,
			Name:     result.TenantName.String,
			Document: result.TenantDocument.String,
		}
		if result.TenantLive.Valid {
			tenant.Live = result.TenantLive.Bool
		}
		if result.TenantCreateAt.Valid {
			tenant.CreateAt = result.TenantCreateAt.Time
		}
		if result.TenantUpdateAt.Valid {
			tenant.UpdateAt = result.TenantUpdateAt.Time
		}
		login.User.Tenant = tenant
	}

	return login, nil
}
