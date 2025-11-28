package middleware

import (
	"tenant-crud-simply/internal/iam/domain/user"
	"time"

	"github.com/google/uuid"
)

type AcessToken struct {
	UserUUID *uuid.UUID `gorm:"type:uuid;index"`
	Token    string     `gorm:"type:varchar(255);not null"`
	Expiry   time.Time  `gorm:"type:timestamp;not null;column:expire_date"`
}

type Login struct {
	User       user.User
	AcessToken AcessToken
}
