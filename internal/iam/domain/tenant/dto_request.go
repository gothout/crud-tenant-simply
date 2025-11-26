package tenant

// CreateTenantRequest representa a requisição para criar um novo tenant
type CreateTenantRequest struct {
	Name     string `json:"name" binding:"required"`
	Document string `json:"document" binding:"required"`
}
