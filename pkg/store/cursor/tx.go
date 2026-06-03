package cursor

import "database/sql"

type Tx interface {
	Commit() error
	Rollback() error

	Query(query string, args ...sql.NamedArg) (*sql.Rows, error)
	QueryRow(query string, args ...sql.NamedArg) *sql.Row
	Exec(query string, args ...sql.NamedArg) (sql.Result, error)
}
