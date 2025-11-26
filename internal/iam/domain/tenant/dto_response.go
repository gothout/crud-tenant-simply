package tenant

import "github.com/google/uuid"

// TenantResponse representa a resposta com os dados de um tenant
type TenantResponse struct {
	UUID     uuid.UUID `json:"uuid"`
	Name     string    `json:"name"`
	Document string    `json:"document"`
	Live     bool      `json:"live"`
}

// ToResponse converte um Tenant para TenantResponse
func ToResponse(tenant Tenant) TenantResponse {
	return TenantResponse{
		UUID:     tenant.UUID,
		Name:     tenant.Name,
		Document: tenant.Document,
		Live:     tenant.Live,
	}
}
