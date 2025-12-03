package user

type CreateUserRequestDto struct {
	Name     string   `json:"name" binding:"required"`
	Email    string   `json:"email" binding:"required,email"`
	Password string   `json:"password" binding:"required,min=8"`
	Role     UserRole `json:"role" binding:"required"`
}

type UpdateUserRequestDto struct {
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Role     UserRole `json:"role"`
}

type ListUserRequestDto struct {
	Page             int    `form:"page"`
	PageSize         int    `form:"size"`
	TenantIdentifier string `form:"tenant_identifier"`
}
