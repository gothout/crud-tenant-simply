package tenant

// CreateTenantRequest representa a requisição para criar um novo tenant
type CreateTenantRequestDto struct {
	Name     string `json:"name" binding:"required"`
	Document string `json:"document" binding:"required"`
}

type ReadTenantRequestDto struct {
	UUID     string `form:"uuid"`
	Document string `form:"document"`
}

type ListTenantRequestDto struct {
	Page     int `form:"page"`
	PageSize int `form:"size"`
}

type UpdateTenantRequestDto struct {
	Name     string `json:"name"`
	Document string `json:"document"`
	Live     *bool  `json:"live"`
}
