package middlewares

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"

	"web-manager/internal/domain"
)

// API purpose (HTTP intent). Used with RequireAuth scopes.
const (
	PurposeRead   = "READ"
	PurposeCreate = "CREATE"
	PurposeUpdate = "UPDATE"
	PurposeDelete = "DELETE"
)

// Logical modules for policy. Expand as you add route groups.
const (
	ModuleUM = "UM" // user management (/admin/users)
)

// Roles allowed to call DELETE-scoped endpoints.
var deleteAllowedRoles = []domain.Role{
	domain.RoleSuperUser,
	domain.RoleManager,
}

// moduleForbiddenRoles: if the user's role is listed, the module is denied entirely.
var moduleForbiddenRoles = map[string][]domain.Role{
	ModuleUM: {domain.RoleAppUser},
}

// moduleForbiddenAppIDs: JWT appId must not be in this list for the given module.
// SUPER_USER ignores these rules (see authScopeError).
var moduleForbiddenAppIDs = map[string][]string{
	// Example: forbid user-admin APIs from a public-facing app id.
	// ModuleUM: {"public-site"},
}

func roleInList(role domain.Role, list []domain.Role) bool {
	return slices.Contains(list, role)
}

func appIDForbiddenForModule(module, tokenAppID string) bool {
	tokenAppID = strings.TrimSpace(tokenAppID)
	if tokenAppID == "" || module == "" {
		return false
	}
	for _, blocked := range moduleForbiddenAppIDs[module] {
		if strings.TrimSpace(blocked) == tokenAppID {
			return true
		}
	}
	return false
}

func moduleForbiddenForRole(module string, role domain.Role) bool {
	if module == "" {
		return false
	}
	return roleInList(role, moduleForbiddenRoles[module])
}

// authScopeError returns a Fiber error if scope rules deny the request; otherwise nil.
// SUPER_USER bypasses module × appId and module × role restrictions (full access).
func authScopeError(scope AuthScope, role domain.Role, tokenAppID string) *fiber.Error {
	if role == domain.RoleSuperUser {
		return nil
	}
	if appIDForbiddenForModule(scope.Module, tokenAppID) {
		return fiber.NewError(fiber.StatusForbidden, "this app is not allowed to use this module")
	}
	if moduleForbiddenForRole(scope.Module, role) {
		return fiber.NewError(fiber.StatusForbidden, "this role cannot access this module")
	}
	p := strings.TrimSpace(strings.ToUpper(scope.Purpose))
	if p == PurposeDelete {
		if !roleInList(role, deleteAllowedRoles) {
			return fiber.NewError(fiber.StatusForbidden, "delete is only allowed for super users and managers")
		}
	}
	return nil
}
