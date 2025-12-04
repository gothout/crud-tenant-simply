package auditoria_log

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uint       `gorm:"primaryKey"`
	TenantUUID *uuid.UUID `gorm:"type:uuid"`
	UserUUID   *uuid.UUID `gorm:"type:uuid"`
	Identifier string     `gorm:"type:text"`

	RayTraceCode string `gorm:"size:100;not null"`

	Domain     string `gorm:"size:100;not null"`
	Action     string `gorm:"size:100;not null"`
	Function   string `gorm:"size:150;not null"`
	Success    bool   `gorm:"not null"`
	InputData  string `gorm:"type:text"`
	OutputData string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (AuditLog) TableName() string {
	return "audit_log"
}
