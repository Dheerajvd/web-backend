package domain

type Role string

const (
	RoleSuperUser Role = "SUPER_USER"
	RoleManager   Role = "MANAGER"
	RoleAppUser   Role = "APP_USER"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)
