package middleware

import (
	"tenant-crud-simply/internal/iam/domain/model"
	"time"

	"github.com/google/uuid"
)

type AcessToken struct {
	UserUUID *uuid.UUID `gorm:"type:uuid;index"`
	Token    string     `gorm:"type:varchar(255);not null"`
	Expiry   time.Time  `gorm:"type:timestamp;not null;column:expire_date"`
}

type Login struct {
	User       model.User
	AcessToken AcessToken
	Metadata   Metadata
}

type Metadata struct {
	RayTraceCode   string
	IP             string
	Agent          string
	Method         string
	Path           string
	Host           string
	Referer        string
	ContentType    string
	UserLanguage   string
	TimeRequest    time.Time
	RequestLatency time.Duration
}
