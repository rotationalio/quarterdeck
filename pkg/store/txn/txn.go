package txn

import (
	"time"

	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// Txn is a storage interface for executing multiple operations against the database so
// that if all operations succeed, the transaction can be committed. If any operation
// fails, the transaction can be rolled back to ensure that the database is not left in
// an inconsistent state. Txn should have similar methods to the Store interface, but
// without requiring the context (this is passed to the transaction when it is created).
type Txn interface {
	Rollback() error
	Commit() error

	UserTxn
	RoleTxn
	PermissionTxn
	APIKeyTxn
	VeroTokenTxn
}

type UserTxn interface {
	ListUsers(*models.UserPage) (*models.UserList, error)
	CreateUser(*models.User) error
	RetrieveUser(id any) (*models.User, error)
	UpdateUser(*models.User) error
	UpdatePassword(ulid.ULID, string) error
	UpdateLastLogin(ulid.ULID, time.Time) error
	DeleteUser(ulid.ULID) error
}

type RoleTxn interface {
	ListRoles(*models.Page) (*models.RoleList, error)
	CreateRole(*models.Role) error
	RetrieveRole(any) (*models.Role, error)
	UpdateRole(*models.Role) error
	AddPermissionToRole(int64, any) error
	RemovePermissionFromRole(int64, int64) error
	DeleteRole(int64) error
}

type PermissionTxn interface {
	ListPermissions(*models.Page) (*models.PermissionList, error)
	CreatePermission(*models.Permission) error
	RetrievePermission(any) (*models.Permission, error)
	UpdatePermission(*models.Permission) error
	DeletePermission(int64) error
}

type APIKeyTxn interface {
	ListAPIKeys(*models.Page) (*models.APIKeyList, error)
	CreateAPIKey(*models.APIKey) error
	RetrieveAPIKey(any) (*models.APIKey, error)
	UpdateAPIKey(*models.APIKey) error
	UpdateLastSeen(ulid.ULID, time.Time) error
	AddPermissionToAPIKey(ulid.ULID, any) error
	RemovePermissionFromAPIKey(ulid.ULID, int64) error
	RevokeAPIKey(ulid.ULID) error
	DeleteAPIKey(ulid.ULID) error
}

type VeroTokenTxn interface {
	CreateVeroToken(*models.VeroToken) error
	RetrieveVeroToken(ulid.ULID) (*models.VeroToken, error)
	UpdateVeroToken(*models.VeroToken) error
	DeleteVeroToken(ulid.ULID) error
	CreateResetPasswordVeroToken(*models.VeroToken) error
}
