package store

import (
	"context"
	"database/sql"
	"io"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/dsn"
	"go.rtnl.ai/quarterdeck/pkg/store/mock"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/quarterdeck/pkg/store/sqlite"
	"go.rtnl.ai/quarterdeck/pkg/store/txn"
	"go.rtnl.ai/ulid"
)

// Open a directory storage provider with the specified URI. Database URLs should either
// specify protocol+transport://user:pass@host/dbname?opt1=a&opt2=b for servers or
// protocol:///relative/path/to/file for embedded databases (for absolute paths, specify
// protocol:////absolute/path/to/file).
func Open(conf config.DatabaseConfig) (s Store, err error) {
	var uri *dsn.DSN
	if uri, err = dsn.Parse(conf.URL); err != nil {
		return nil, err
	}

	// The configuration overrides any read-only setting in the DSN.
	uri.ReadOnly = conf.ReadOnly

	switch uri.Scheme {
	case dsn.Mock:
		return mock.Open(uri)
	case dsn.SQLite, dsn.SQLite3:
		return sqlite.Open(uri)
	default:
		return nil, errors.Fmt("unhandled database scheme %q", uri.Scheme)
	}
}

// Store is a generic storage interface allowing multiple storage backends such as
// SQLite or Postgres to be used based on the preference of the user.
// NOTE: to prevent import cycles, the txn.Tx interface is in its own package. If an
// interface is added to the Store interface, it should be added to the txn.Tx interface
// as well (to ensure the Txn has the same methods as the Store).
type Store interface {
	Begin(context.Context, *sql.TxOptions) (txn.Txn, error)

	io.Closer
	UserStore
	RoleStore
	PermissionStore
	APIKeyStore
	VeroTokenStore
}

// The Stats interface exposes database statistics if it is available from the backend.
type Stats interface {
	Stats() sql.DBStats
}

type UserStore interface {
	ListUsers(context.Context, *models.UserPage) (*models.UserList, error)
	CreateUser(context.Context, *models.User) error
	RetrieveUser(context.Context, any) (*models.User, error)
	UpdateUser(context.Context, *models.User) error
	UpdatePassword(context.Context, ulid.ULID, string) error
	UpdateLastLogin(context.Context, ulid.ULID, time.Time) error
	VerifyEmail(context.Context, ulid.ULID) error
	DeleteUser(context.Context, ulid.ULID) error
}

type RoleStore interface {
	ListRoles(context.Context, *models.Page) (*models.RoleList, error)
	CreateRole(context.Context, *models.Role) error
	RetrieveRole(context.Context, any) (*models.Role, error)
	UpdateRole(context.Context, *models.Role) error
	AddPermissionToRole(context.Context, int64, any) error
	RemovePermissionFromRole(context.Context, int64, int64) error
	DeleteRole(context.Context, int64) error
}

type PermissionStore interface {
	ListPermissions(context.Context, *models.Page) (*models.PermissionList, error)
	CreatePermission(context.Context, *models.Permission) error
	RetrievePermission(context.Context, any) (*models.Permission, error)
	UpdatePermission(context.Context, *models.Permission) error
	DeletePermission(context.Context, int64) error
}

type APIKeyStore interface {
	ListAPIKeys(context.Context, *models.Page) (*models.APIKeyList, error)
	CreateAPIKey(context.Context, *models.APIKey) error
	RetrieveAPIKey(context.Context, any) (*models.APIKey, error)
	UpdateAPIKey(context.Context, *models.APIKey) error
	UpdateLastSeen(context.Context, ulid.ULID, time.Time) error
	AddPermissionToAPIKey(context.Context, ulid.ULID, any) error
	RemovePermissionFromAPIKey(context.Context, ulid.ULID, int64) error
	RevokeAPIKey(context.Context, ulid.ULID) error
	DeleteAPIKey(context.Context, ulid.ULID) error
}

type VeroTokenStore interface {
	CreateVeroToken(context.Context, *models.VeroToken) error
	RetrieveVeroToken(context.Context, ulid.ULID) (*models.VeroToken, error)
	UpdateVeroToken(context.Context, *models.VeroToken) error
	DeleteVeroToken(context.Context, ulid.ULID) error
	CreateResetPasswordVeroToken(context.Context, *models.VeroToken) error
}
