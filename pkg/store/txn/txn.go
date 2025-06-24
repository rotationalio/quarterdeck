package txn

// Txn is a storage interface for executing multiple operations against the database so
// that if all operations succeed, the transaction can be committed. If any operation
// fails, the transaction can be rolled back to ensure that the database is not left in
// an inconsistent state. Txn should have similar methods to the Store interface, but
// without requiring the context (this is passed to the transaction when it is created).
type Txn interface {
	Rollback() error
	Commit() error
}
