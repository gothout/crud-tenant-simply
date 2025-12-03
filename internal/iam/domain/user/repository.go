package user

import (
	"context"
	"errors"
	"fmt"
	"tenant-crud-simply/internal/iam/domain/tenant"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, user User) (User, error)
	Read(ctx context.Context, user User) (User, error)
	List(ctx context.Context, page, pageSize int) ([]User, error)
	ListByTenant(ctx context.Context, tenant tenant.Tenant, page, pageSize int) ([]User, error)
	Update(ctx context.Context, user User) (User, error)
	Delete(ctx context.Context, user User) error
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

func (r *repositoryImpl) Read(ctx context.Context, user User) (User, error) {
	query := r.db.WithContext(ctx).Last(&User{})
	if user.UUID != uuid.Nil {
		query = query.Where("uuid = ?", user.UUID).First(&user)
	} else if user.Email != "" {
		query = query.Where("email = ?", user.Email).First(&user)
	} else {
		return User{}, ErrInvalidInput
	}
	if query.Error != nil {
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return User{}, ErrNotFound
		}
		return User{}, query.Error
	}
	return user, nil
}

func (r *repositoryImpl) List(ctx context.Context, page, pageSize int) ([]User, error) {
	var users []User
	query := r.db.WithContext(ctx).Model(&users)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	result := query.Limit(pageSize).Offset(offset).Find(&users)
	if result.Error != nil {
		return users, result.Error
	}
	return users, nil
}

func (r *repositoryImpl) ListByTenant(ctx context.Context, t tenant.Tenant, page, pageSize int) ([]User, error) {
	var users []User

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	query := r.db.WithContext(ctx).Model(&User{})
	if t.UUID != uuid.Nil {
		query = query.Where("tenant_uuid = ?", t.UUID)

	} else if t.Document != "" {
		query = query.Joins("INNER JOIN tenant ON tenant.uuid = users.tenant_uuid").
			Where("tenant.document = ?", t.Document)

	} else {
		return nil, errors.New("é necessário informar o UUID ou o Documento do Tenant")
	}
	result := query.Limit(pageSize).Offset(offset).Find(&users)

	if result.Error != nil {
		return users, result.Error
	}

	return users, nil
}

func (r *repositoryImpl) Update(ctx context.Context, user User) (User, error) {
	updateFields := make(map[string]interface{})

	// Monta apenas os campos enviados
	if user.Name != "" {
		updateFields["name"] = user.Name
	}
	if user.Email != "" {
		updateFields["email"] = user.Email
	}
	if user.Password != "" {
		updateFields["password_hash"] = user.Password
	}
	if user.Role != "" {
		updateFields["role"] = user.Role
	}
	// Cuidado: se Live for false, ele não entra aqui (a menos que use ponteiro)
	if user.Live {
		updateFields["live"] = user.Live
	}
	if !user.UpdateAt.IsZero() {
		updateFields["update_at"] = user.UpdateAt
	}

	if len(updateFields) == 0 {
		return User{}, errors.New("nenhum campo válido para atualização")
	}

	// Executa o update
	query := r.db.WithContext(ctx).
		Model(&User{}).
		Where("uuid = ?", user.UUID).
		Updates(updateFields)

	// --- Tratamento de erros do PostgreSQL ---
	if query.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(query.Error, &pgErr) {
			switch pgErr.Code {
			case "23505": // Unique violation
				if pgErr.ConstraintName == "users_email_key" {
					return User{}, ErrEmailDuplicated
				}
				return User{}, fmt.Errorf("violação de unicidade (%s): %w", pgErr.ConstraintName, query.Error)
			case "23503": // Foreign key violation
				return User{}, tenant.ErrNotFound
			default:
				return User{}, fmt.Errorf("erro do banco (%s): %w", pgErr.Code, query.Error)
			}
		}
		// Erro genérico (não é PgError)
		return User{}, query.Error
	}

	// --- Verifica se o registro realmente existe ---
	if query.RowsAffected == 0 {
		existingUser, err := r.Read(ctx, User{UUID: user.UUID})
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return User{}, ErrNotFound
			}
			return User{}, err
		}
		return existingUser, nil
	}

	// --- Retorna o usuário atualizado ---
	updatedUser, err := r.Read(ctx, User{UUID: user.UUID})
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return updatedUser, nil
}

func (r *repositoryImpl) Delete(ctx context.Context, user User) error {
	query := r.db.WithContext(ctx).Where("uuid = ?", user.UUID).Delete(&User{})
	if query.Error != nil {
		return query.Error
	}
	if query.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
