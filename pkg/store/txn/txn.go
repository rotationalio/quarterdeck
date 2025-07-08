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
