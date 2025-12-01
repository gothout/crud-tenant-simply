package model

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleSystemAdmin UserRole = "SYSTEM_ADMIN"
	RoleTenantAdmin UserRole = "TENANT_ADMIN"
	RoleTenantUser  UserRole = "TENANT_USER"
)

type User struct {
	UUID       uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantUUID *uuid.UUID `gorm:"type:uuid;index"`
	Name       string     `gorm:"type:varchar(255);not null"`
	Email      string     `gorm:"type:varchar(255);not null;unique"`
	Password   string     `gorm:"column:password_hash;type:varchar(255);not null"`
	Role       UserRole   `gorm:"type:user_role;not null;default:'TENANT_USER'"`
	Live       bool       `gorm:"not null;default:true"`
	CreateAt   time.Time  `gorm:"column:create_at;not null;autoCreateTime"`
	UpdateAt   time.Time  `gorm:"column:update_at;not null;autoUpdateTime"`
	Tenant     Tenant     `gorm:"foreignKey:TenantUUID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}
