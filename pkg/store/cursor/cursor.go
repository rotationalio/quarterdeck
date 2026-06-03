package cursor

import (
	"database/sql"

	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

type Cursor[M models.Model] interface {
	Next() bool
	Model() (M, error)
	List() ([]M, error)
	Close() error
	Err() error
}

//============================================================================
// SQL Rows Cursor
//============================================================================

func Rows[M models.Model](tx Tx, rows *sql.Rows) Cursor[M] {
	return &sqlCursor[M]{tx: tx, rows: rows}
}

type sqlCursor[M models.Model] struct {
	tx   Tx
	rows *sql.Rows
}

func (c *sqlCursor[M]) Next() bool {
	return c.rows.Next()
}

func (c *sqlCursor[M]) Model() (M, error) {
	model := models.Make[M]()
	if err := model.Scan(models.List, c.rows); err != nil {
		return model, err
	}
	return model, nil
}

func (c *sqlCursor[M]) List() (models []M, err error) {
	models = make([]M, 0)
	for c.Next() {
		var model M
		if model, err = c.Model(); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, c.Err()
}

func (c *sqlCursor[M]) Close() error {
	c.tx.Rollback()
	return c.rows.Close()
}

func (c *sqlCursor[M]) Err() error {
	return c.rows.Err()
}

//============================================================================
// Empty Cursor
//============================================================================

func Err[M models.Model](err error) Cursor[M] {
	return &emptyCursor[M]{err: err}
}

type emptyCursor[M models.Model] struct {
	err error
}

func (c *emptyCursor[M]) Next() bool {
	return false
}

func (c *emptyCursor[M]) Model() (M, error) {
	var model M
	return model, c.err
}

func (c *emptyCursor[M]) List() ([]M, error) {
	return nil, c.err
}

func (c *emptyCursor[M]) Close() error {
	return c.err
}

func (c *emptyCursor[M]) Err() error {
	return c.err
}
