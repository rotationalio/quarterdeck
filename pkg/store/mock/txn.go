package mock

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// Transaction method names
const (
	Commit   = "Commit"
	Rollback = "Rollback"
)

type Tx struct {
	opts     *sql.TxOptions
	calls    map[string]int
	commit   bool
	rollback bool

	// Txn callbacks
	OnCommit   func() error
	OnRollback func() error

	// UserTxn Callbacks
	OnListUsers       func(*models.UserPage) (*models.UserList, error)
	OnCreateUser      func(*models.User) error
	OnRetrieveUser    func(any) (*models.User, error)
	OnUpdateUser      func(*models.User) error
	OnUpdatePassword  func(ulid.ULID, string) error
	OnUpdateLastLogin func(ulid.ULID, time.Time) error
	OnVerifyEmail     func(ulid.ULID) error
	OnDeleteUser      func(ulid.ULID) error

	// RoleTxn Callbacks
	OnListRoles                func(*models.Page) (*models.RoleList, error)
	OnCreateRole               func(*models.Role) error
	OnRetrieveRole             func(any) (*models.Role, error)
	OnUpdateRole               func(*models.Role) error
	OnAddPermissionToRole      func(int64, any) error
	OnRemovePermissionFromRole func(int64, int64) error
	OnDeleteRole               func(int64) error

	// PermissionTxn Callbacks
	OnListPermissions    func(*models.Page) (*models.PermissionList, error)
	OnCreatePermission   func(*models.Permission) error
	OnRetrievePermission func(any) (*models.Permission, error)
	OnUpdatePermission   func(*models.Permission) error
	OnDeletePermission   func(int64) error

	// APIKeyTxn Callbacks
	OnListAPIKeys                func(*models.Page) (*models.APIKeyList, error)
	OnCreateAPIKey               func(*models.APIKey) error
	OnRetrieveAPIKey             func(any) (*models.APIKey, error)
	OnUpdateAPIKey               func(*models.APIKey) error
	OnUpdateLastSeen             func(ulid.ULID, time.Time) error
	OnAddPermissionToAPIKey      func(ulid.ULID, any) error
	OnRemovePermissionFromAPIKey func(ulid.ULID, int64) error
	OnRevokeAPIKey               func(ulid.ULID) error
	OnDeleteAPIKey               func(ulid.ULID) error

	// VeroTokenTxn Callbacks
	OnCreateVeroToken              func(*models.VeroToken) error
	OnRetrieveVeroToken            func(ulid.ULID) (*models.VeroToken, error)
	OnUpdateVeroToken              func(*models.VeroToken) error
	OnDeleteVeroToken              func(ulid.ULID) error
	OnCreateResetPasswordVeroToken func(*models.VeroToken) error
	OnCreateTeamInviteVeroToken    func(*models.VeroToken) error

	// Application callbacks
	OnListApplications    func(*models.Page) (*models.ApplicationList, error)
	OnCreateApplication   func(*models.Application) error
	OnRetrieveApplication func(ulidOrClientID any) (*models.Application, error)
	OnUpdateApplication   func(*models.Application) error
	OnDeleteApplication   func(ulidOrClientID any) error
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the store.
func (tx *Tx) Reset() {
	// reset the call counts
	tx.calls = nil
	tx.calls = make(map[string]int)

	// reset the callbacks using reflection
	v := reflect.ValueOf(tx).Elem()
	t := v.Type()
	for _, f := range reflect.VisibleFields(t) {
		// only reset functions named `OnSomething`
		if strings.HasPrefix(f.Name, "On") && f.Type.Kind() == reflect.Func {
			fv := v.FieldByIndex(f.Index)
			fv.SetZero()
		}
	}
}

// Assert that the expected number of calls were made to the given method.
func (tx *Tx) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, tx.calls[method], "expected %d calls to %s, got %d", expected, method, tx.calls[method])
}

// Assert that Commit has been called on the transaction without rollback.
func (tx *Tx) AssertCommit(t testing.TB) {
	require.True(t, tx.commit && !tx.rollback, "expected Commit to be called but not Rollback")
}

// Assert that Rollback has been called on the transaction without commit.
func (tx *Tx) AssertRollback(t testing.TB) {
	require.True(t, tx.rollback && !tx.commit, "expected Rollback to be called but not Commit")
}

// Assert that Commit has not been called on the transaction.
func (tx *Tx) AssertNoCommit(t testing.TB) {
	require.False(t, tx.commit, "did not expect Commit to be called")
}

// Assert that Rollback has not been called on the transaction.
func (tx *Tx) AssertNoRollback(t testing.TB) {
	require.False(t, tx.rollback, "did not expect Rollback to be called")
}

// Check is a helper method that determines if the transaction is committed or rolled
// back. If so it returns ErrTxDone no matter the callback. Additionally, if the method
// is writeable and the transaction is read-only, it returns an error. This method also
// increments the call count for the method.
func (tx *Tx) check(method string, writeable bool) error {
	tx.calls[method]++

	if tx.commit || tx.rollback {
		return sql.ErrTxDone
	}

	if tx.opts != nil && tx.opts.ReadOnly && writeable {
		return errors.ErrReadOnly
	}

	return nil
}

//===========================================================================
// Transaction Base Methods
//===========================================================================

func (tx *Tx) Commit() (err error) {
	if err = tx.check(Commit, false); err != nil {
		return err
	}

	if tx.OnCommit != nil {
		err = tx.OnCommit()
	}

	tx.commit = true
	return err
}

func (tx *Tx) Rollback() (err error) {
	if err := tx.check(Rollback, false); err != nil {
		return err
	}

	if tx.OnRollback != nil {
		err = tx.OnRollback()
	}

	tx.rollback = true
	return err
}

//===========================================================================
// UserTxn Methods
//===========================================================================

func (tx *Tx) ListUsers(page *models.UserPage) (*models.UserList, error) {
	tx.calls[ListUsers]++
	if tx.OnListUsers != nil {
		return tx.OnListUsers(page)
	}
	panic(errors.Fmt("%s callback is not mocked", ListUsers))
}

func (tx *Tx) CreateUser(user *models.User) error {
	tx.calls[CreateUser]++
	if tx.OnCreateUser != nil {
		return tx.OnCreateUser(user)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateUser))
}

func (tx *Tx) RetrieveUser(id any) (*models.User, error) {
	tx.calls[RetrieveUser]++
	if tx.OnRetrieveUser != nil {
		return tx.OnRetrieveUser(id)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveUser))
}

func (tx *Tx) UpdateUser(user *models.User) error {
	tx.calls[UpdateUser]++
	if tx.OnUpdateUser != nil {
		return tx.OnUpdateUser(user)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateUser))
}

func (tx *Tx) UpdatePassword(id ulid.ULID, password string) error {
	tx.calls[UpdatePassword]++
	if tx.OnUpdatePassword != nil {
		return tx.OnUpdatePassword(id, password)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdatePassword))
}

func (tx *Tx) UpdateLastLogin(id ulid.ULID, lastLogin time.Time) error {
	tx.calls[UpdateLastLogin]++
	if tx.OnUpdateLastLogin != nil {
		return tx.OnUpdateLastLogin(id, lastLogin)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateLastLogin))
}

func (tx *Tx) VerifyEmail(id ulid.ULID) error {
	tx.calls[VerifyEmail]++
	if tx.OnVerifyEmail != nil {
		return tx.OnVerifyEmail(id)
	}
	panic(errors.Fmt("%s callback is not mocked", VerifyEmail))
}

func (tx *Tx) DeleteUser(id ulid.ULID) error {
	tx.calls[DeleteUser]++
	if tx.OnDeleteUser != nil {
		return tx.OnDeleteUser(id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteUser))
}

//===========================================================================
// RoleTxn Methods
//===========================================================================

func (tx *Tx) ListRoles(in *models.Page) (*models.RoleList, error) {
	tx.calls[ListRoles]++
	if tx.OnListRoles != nil {
		return tx.OnListRoles(in)
	}
	panic(errors.Fmt("%s callback is not mocked", ListRoles))
}

func (tx *Tx) CreateRole(role *models.Role) error {
	tx.calls[CreateRole]++
	if tx.OnCreateRole != nil {
		return tx.OnCreateRole(role)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateRole))
}

func (tx *Tx) RetrieveRole(in any) (*models.Role, error) {
	tx.calls[RetrieveRole]++
	if tx.OnRetrieveRole != nil {
		return tx.OnRetrieveRole(in)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveRole))
}

func (tx *Tx) UpdateRole(role *models.Role) error {
	tx.calls[UpdateRole]++
	if tx.OnUpdateRole != nil {
		return tx.OnUpdateRole(role)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateRole))
}

func (tx *Tx) AddPermissionToRole(roleID int64, permissionID any) error {
	tx.calls[AddPermissionToRole]++
	if tx.OnAddPermissionToRole != nil {
		return tx.OnAddPermissionToRole(roleID, permissionID)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToRole))
}

func (tx *Tx) RemovePermissionFromRole(roleID int64, permissionID int64) error {
	tx.calls[RemovePermissionFromRole]++
	if tx.OnRemovePermissionFromRole != nil {
		return tx.OnRemovePermissionFromRole(roleID, permissionID)
	}
	panic(errors.Fmt("%s callback is not mocked", RemovePermissionFromRole))
}

func (tx *Tx) DeleteRole(id int64) error {
	tx.calls[DeleteRole]++
	if tx.OnDeleteRole != nil {
		return tx.OnDeleteRole(id)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteRole))
}

//===========================================================================
// PermissionTxn Methods
//===========================================================================

func (tx *Tx) ListPermissions(in *models.Page) (*models.PermissionList, error) {
	tx.calls[ListPermissions]++
	if tx.OnListPermissions != nil {
		return tx.OnListPermissions(in)
	}
	panic(errors.Fmt("%s callback is not mocked", ListPermissions))
}

func (tx *Tx) CreatePermission(in *models.Permission) error {
	tx.calls[CreatePermission]++
	if tx.OnCreatePermission != nil {
		return tx.OnCreatePermission(in)
	}
	panic(errors.Fmt("%s callback is not mocked", CreatePermission))
}

func (tx *Tx) RetrievePermission(in any) (*models.Permission, error) {
	tx.calls[RetrievePermission]++
	if tx.OnRetrievePermission != nil {
		return tx.OnRetrievePermission(in)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrievePermission))
}

func (tx *Tx) UpdatePermission(in *models.Permission) error {
	tx.calls[UpdatePermission]++
	if tx.OnUpdatePermission != nil {
		return tx.OnUpdatePermission(in)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdatePermission))
}

func (tx *Tx) DeletePermission(in int64) error {
	tx.calls[DeletePermission]++
	if tx.OnDeletePermission != nil {
		return tx.OnDeletePermission(in)
	}
	panic(errors.Fmt("%s callback is not mocked", DeletePermission))
}

//===========================================================================
// APIKeyTxn Methods
//===========================================================================

func (tx *Tx) ListAPIKeys(in *models.Page) (*models.APIKeyList, error) {
	tx.calls[ListAPIKeys]++
	if tx.OnListAPIKeys != nil {
		return tx.OnListAPIKeys(in)
	}
	panic(errors.Fmt("%s callback is not mocked", ListAPIKeys))
}

func (tx *Tx) CreateAPIKey(in *models.APIKey) error {
	tx.calls[CreateAPIKey]++
	if tx.OnCreateAPIKey != nil {
		return tx.OnCreateAPIKey(in)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateAPIKey))
}

func (tx *Tx) RetrieveAPIKey(in any) (*models.APIKey, error) {
	tx.calls[RetrieveAPIKey]++
	if tx.OnRetrieveAPIKey != nil {
		return tx.OnRetrieveAPIKey(in)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveAPIKey))
}

func (tx *Tx) UpdateAPIKey(in *models.APIKey) error {
	tx.calls[UpdateAPIKey]++
	if tx.OnUpdateAPIKey != nil {
		return tx.OnUpdateAPIKey(in)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateAPIKey))
}

func (tx *Tx) UpdateLastSeen(id ulid.ULID, lastSeen time.Time) error {
	tx.calls[UpdateLastSeen]++
	if tx.OnUpdateLastSeen != nil {
		return tx.OnUpdateLastSeen(id, lastSeen)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateLastSeen))
}

func (tx *Tx) AddPermissionToAPIKey(id ulid.ULID, permission any) error {
	tx.calls[AddPermissionToAPIKey]++
	if tx.OnAddPermissionToAPIKey != nil {
		return tx.OnAddPermissionToAPIKey(id, permission)
	}
	panic(errors.Fmt("%s callback is not mocked", AddPermissionToAPIKey))
}

func (tx *Tx) RemovePermissionFromAPIKey(id ulid.ULID, permission int64) error {
	tx.calls[RemovePermissionFromAPIKey]++
	if tx.OnRemovePermissionFromAPIKey != nil {
		return tx.OnRemovePermissionFromAPIKey(id, permission)
	}
	panic(errors.Fmt("%s callback is not mocked", RemovePermissionFromAPIKey))
}

func (tx *Tx) RevokeAPIKey(in ulid.ULID) error {
	tx.calls[RevokeAPIKey]++
	if tx.OnRevokeAPIKey != nil {
		return tx.OnRevokeAPIKey(in)
	}
	panic(errors.Fmt("%s callback is not mocked", RevokeAPIKey))
}

func (tx *Tx) DeleteAPIKey(in ulid.ULID) error {
	tx.calls[DeleteAPIKey]++
	if tx.OnDeleteAPIKey != nil {
		return tx.OnDeleteAPIKey(in)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteAPIKey))
}

//===========================================================================
// VeroTokenTxn Methods
//===========================================================================

func (tx *Tx) CreateVeroToken(in *models.VeroToken) error {
	tx.calls[CreateVeroToken]++
	if tx.OnCreateVeroToken != nil {
		return tx.OnCreateVeroToken(in)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateVeroToken))
}

func (tx *Tx) RetrieveVeroToken(in ulid.ULID) (*models.VeroToken, error) {
	tx.calls[RetrieveVeroToken]++
	if tx.OnRetrieveVeroToken != nil {
		return tx.OnRetrieveVeroToken(in)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveVeroToken))
}

func (tx *Tx) UpdateVeroToken(in *models.VeroToken) error {
	tx.calls[UpdateVeroToken]++
	if tx.OnUpdateVeroToken != nil {
		return tx.OnUpdateVeroToken(in)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateVeroToken))
}

func (tx *Tx) DeleteVeroToken(in ulid.ULID) error {
	tx.calls[DeleteVeroToken]++
	if tx.OnDeleteVeroToken != nil {
		return tx.OnDeleteVeroToken(in)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteVeroToken))
}

func (tx *Tx) CreateResetPasswordVeroToken(in *models.VeroToken) error {
	tx.calls[CreateResetPasswordVeroToken]++
	if tx.OnCreateResetPasswordVeroToken != nil {
		return tx.OnCreateResetPasswordVeroToken(in)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateResetPasswordVeroToken))
}

func (tx *Tx) CreateTeamInviteVeroToken(in *models.VeroToken) error {
	tx.calls[CreateTeamInviteVeroToken]++
	if tx.OnCreateTeamInviteVeroToken != nil {
		return tx.OnCreateTeamInviteVeroToken(in)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateResetPasswordVeroToken))
}

//===========================================================================
// ApplicationStore
//===========================================================================

func (t *Tx) ListApplications(in *models.Page) (*models.ApplicationList, error) {
	t.calls[ListApplications]++
	if t.OnListApplications != nil {
		return t.OnListApplications(in)
	}
	panic(errors.Fmt("%s callback is not mocked", ListApplications))
}

func (t *Tx) CreateApplication(in *models.Application) error {
	t.calls[CreateApplication]++
	if t.OnCreateApplication != nil {
		return t.OnCreateApplication(in)
	}
	panic(errors.Fmt("%s callback is not mocked", CreateApplication))
}

func (t *Tx) RetrieveApplication(in any) (*models.Application, error) {
	t.calls[RetrieveApplication]++
	if t.OnRetrieveApplication != nil {
		return t.OnRetrieveApplication(in)
	}
	panic(errors.Fmt("%s callback is not mocked", RetrieveApplication))
}

func (t *Tx) UpdateApplication(in *models.Application) error {
	t.calls[UpdateApplication]++
	if t.OnUpdateApplication != nil {
		return t.OnUpdateApplication(in)
	}
	panic(errors.Fmt("%s callback is not mocked", UpdateApplication))
}

func (t *Tx) DeleteApplication(in any) error {
	t.calls[DeleteApplication]++
	if t.OnDeleteApplication != nil {
		return t.OnDeleteApplication(in)
	}
	panic(errors.Fmt("%s callback is not mocked", DeleteApplication))
}
