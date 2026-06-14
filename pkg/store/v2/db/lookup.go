package db

import (
	"database/sql"

	"go.rtnl.ai/tidal"
)

// retrieveBy retrieves a model of type M by a single column name and value.
// Returns an error if the record is not found or if there is a database error.
//
// Example usage:
//
//	apiKey, err := retrieveBy[*models.APIKey](tx, apiKeys, "client_id", "DtptIgWgzkwaibktjczVwr")
//	if err != nil {
//	    return nil, err
//	}
//	return apiKey, nil
func retrieveBy[M tidal.Model](tx tidal.Tx, crud *tidal.CRUD[M], column string, value any) (m M, err error) {
	m = tidal.Make[M]()
	query := crud.Queries.Retrieve + column + " = :" + column
	if err = m.Scan(tidal.Retrieve, tx.QueryRow(query, sql.Named(column, value))); err != nil {
		return m, tidalErr(err)
	}
	return m, nil
}
