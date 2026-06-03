package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/cursor"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

// A QuerySet is the precomputed SQL query strings for a given model. These are
// computed at initialization to avoid repeated string templating for every query.
type QuerySet struct {
	List     string // Must be ready to attach filter clauses using concatenation.
	Create   string // Will not be edited by the CRUD model.
	Retrieve string // Must not contain a WHERE clause.
	Update   string // Will not be edited by the CRUD model.
	Delete   string // Must not contain a WHERE clause.
}

type CRUD[M models.Model] struct {
	Queries QuerySet
}

func MakeCRUD[M models.Model](table string) *CRUD[M] {
	return &CRUD[M]{
		Queries: QuerySet{
			List:     ListQuery[M](table),
			Create:   CreateQuery[M](table),
			Retrieve: RetrieveQuery[M](table),
			Update:   UpdateQuery[M](table, "id"),
			Delete:   DeleteQuery[M](table),
		},
	}
}

func (c *CRUD[M]) List(tx cursor.Tx, filter cursor.Filter) (_ cursor.Cursor[M], err error) {
	// Add the filtering constraints to the query (if any).
	var params []sql.NamedArg
	query := c.Queries.List

	if filter != nil {
		if clause := filter.SQL(); clause != "" {
			query += " " + clause
		}
		params = filter.Params()
	}

	var rows *sql.Rows
	if rows, err = tx.Query(query, params...); err != nil {
		return nil, dbe(err)
	}

	return cursor.Rows[M](tx, rows), nil
}

func (c *CRUD[M]) Create(tx cursor.Tx, m M) (result sql.Result, err error) {
	if prepare, ok := any(m).(models.Preparer); ok {
		prepare.Prepare(models.Create)
	}

	if validator, ok := any(m).(models.Validator); ok {
		if err = validator.Validate(models.Create); err != nil {
			return nil, err
		}
	}

	if result, err = tx.Exec(c.Queries.Create, m.Params(models.Create)...); err != nil {
		return nil, dbe(err)
	}

	return result, nil
}

func (c *CRUD[M]) Retrieve(tx cursor.Tx, id sql.NamedArg) (m M, err error) {
	m = models.Make[M]()
	query := c.Queries.Retrieve + id.Name + " = :" + id.Name
	if err = m.Scan(models.Retrieve, tx.QueryRow(query, id)); err != nil {
		return m, dbe(err)
	}
	return m, nil
}

func (c *CRUD[M]) Update(tx cursor.Tx, m M) (err error) {
	if prepare, ok := any(m).(models.Preparer); ok {
		prepare.Prepare(models.Update)
	}

	if validator, ok := any(m).(models.Validator); ok {
		if err = validator.Validate(models.Update); err != nil {
			return err
		}
	}

	var result sql.Result
	if result, err = tx.Exec(c.Queries.Update, m.Params(models.Update)...); err != nil {
		return dbe(err)
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}
	return nil
}

func (c *CRUD[M]) Delete(tx cursor.Tx, id sql.NamedArg) (result sql.Result, err error) {
	query := c.Queries.Delete + id.Name + " = :" + id.Name
	if result, err = tx.Exec(query, id); err != nil {
		return nil, dbe(err)
	}
	return result, nil
}

func Fields[M models.Model](op models.Operation) []string {
	var m M
	return m.Fields(op)
}

func Params[M models.Model](op models.Operation) (fields []string, placeholders []string) {
	m := models.Make[M]()
	params := m.Params(op)

	fields = make([]string, len(params))
	placeholders = make([]string, len(params))

	for i, param := range params {
		fields[i] = param.Name
		placeholders[i] = ":" + param.Name
	}
	return fields, placeholders
}

func ListQuery[M models.Model](table string) string {
	return fmt.Sprintf("SELECT %s FROM %s", strings.Join(Fields[M](models.List), ", "), table)
}

func CreateQuery[M models.Model](table string) string {
	fields, placeholders := Params[M](models.Create)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(fields, ", "), strings.Join(placeholders, ", "))
}

func RetrieveQuery[M models.Model](table string) string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE ", strings.Join(Fields[M](models.Retrieve), ", "), table)
}

func UpdateQuery[M models.Model](table string, fieldID string) string {
	fields, placeholders := Params[M](models.Update)
	setters := make([]string, 0, len(fields))

	for i, field := range fields {
		if field == fieldID {
			continue
		}
		setters = append(setters, fmt.Sprintf("%s=%s", field, placeholders[i]))
	}

	return fmt.Sprintf("UPDATE %s SET %s WHERE %s=:%s", table, strings.Join(setters, ", "), fieldID, fieldID)
}

func DeleteQuery[M models.Model](table string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE ", table)
}
