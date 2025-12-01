package model

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	UUID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name     string    `gorm:"type:varchar(255);not null"`
	Document string    `gorm:"type:varchar(100);not null;unique"`
	Live     bool      `gorm:"type:boolean;not null;default:true"`
	CreateAt time.Time `gorm:"type:timestamp without time zone;not null"`
	UpdateAt time.Time `gorm:"type:timestamp without time zone;not null"`
}

func (Tenant) TableName() string {
	return "tenant"
}
