package user

import (
	"tenant-crud-simply/internal/iam/domain/model"
)

// --- Type Aliases ---
type User = model.User
type UserRole = model.UserRole

const (
	RoleSystemAdmin = model.RoleSystemAdmin
	RoleTenantAdmin = model.RoleTenantAdmin
	RoleTenantUser  = model.RoleTenantUser
)

var validRolesMap = map[UserRole]bool{
	RoleSystemAdmin: true,
	RoleTenantAdmin: true,
	RoleTenantUser:  true,
}

var AllValidRoles = []string{
	"SYSTEM_ADMIN",
	"TENANT_ADMIN",
	"TENANT_USER",
}

func IsValidUserRole(r UserRole) bool {
	_, ok := validRolesMap[r]
	return ok
}
