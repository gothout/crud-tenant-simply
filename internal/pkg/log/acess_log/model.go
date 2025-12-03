package acess_log

import (
	"time"

	"github.com/google/uuid"
)

type AccessLog struct {
	ID         uint       `gorm:"primaryKey"`
	TenantUUID *uuid.UUID `gorm:"type:uuid"`
	UserUUID   *uuid.UUID `gorm:"type:uuid"`
	Identifier string     `gorm:"type:text"`

	RayTraceCode string `gorm:"size:100;not null"`

	Method       string `gorm:"size:10;not null"`
	Path         string `gorm:"type:text;not null"`
	Host         string `gorm:"type:text;not null"`
	StatusCode   int    `gorm:"not null"`
	IP           string `gorm:"type:inet;not null"`
	UserAgent    string `gorm:"type:text"`
	Referer      string `gorm:"type:text"`
	ContentType  string `gorm:"type:text"`
	UserLanguage string `gorm:"type:text"`

	RequestTime time.Time `gorm:"not null"`
	LatencyMs   float64   `gorm:"not null"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (AccessLog) TableName() string {
	return "access_log"
}
