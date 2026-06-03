package cursor

import "database/sql"

type Filter interface {
	SQL() string
	Params() []sql.NamedArg
}
