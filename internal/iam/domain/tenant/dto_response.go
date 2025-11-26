package tenant

import (
	"time"

	"github.com/google/uuid"
)

// TenantResponse representa a resposta com os dados de um tenant
type TenantResponseDto struct {
	UUID     uuid.UUID `json:"uuid"`
	Name     string    `json:"name"`
	Document string    `json:"document"`
	Live     bool      `json:"live"`
	CreateAt time.Time `json:"createAt"`
	UpdateAt time.Time `json:"updateAt"`
}

type TenantsResponseDto struct {
	Tenants []TenantResponseDto `json:"tenants"`
	Page    int                 `json:"page"`
	Size    int                 `json:"size"`
}
