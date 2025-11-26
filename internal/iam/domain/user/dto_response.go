package user

import (
	"time"

	"github.com/google/uuid"
)

type UserResponseDto struct {
	UUID       uuid.UUID  `json:"uuid"`
	TenantUUID *uuid.UUID `json:"tenant_uuid,omitempty"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Role       UserRole   `json:"role"`
	Live       bool       `json:"live"`
	CreateAt   time.Time  `json:"create_at"`
	UpdateAt   time.Time  `json:"update_at"`
}

type UserListResponseDto struct {
	Users []UserResponseDto `json:"users"`
}
