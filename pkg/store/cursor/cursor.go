package cursor

import "go.rtnl.ai/quarterdeck/pkg/store/models"

type Cursor[M models.Model] interface {
	Next() bool
	Model() (M, error)
	Close() error
	Err() error
}
