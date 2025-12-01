package user

import "tenant-crud-simply/internal/iam/domain/model"

type User = model.User
type UserRole = model.UserRole

const (
	RoleSystemAdmin = model.RoleSystemAdmin
	RoleTenantAdmin = model.RoleTenantAdmin
	RoleTenantUser  = model.RoleTenantUser
)
