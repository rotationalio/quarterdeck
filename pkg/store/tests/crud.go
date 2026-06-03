package tests

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/quarterdeck/pkg/store/cursor"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
)

// Interface for CRUD implementations.
type CRUD[m models.Model] interface {
	List(tx cursor.Tx, filter cursor.Filter) (cursor.Cursor[m], error)
	Create(tx cursor.Tx, m m) (sql.Result, error)
	Retrieve(tx cursor.Tx, id sql.NamedArg) (m, error)
	Update(tx cursor.Tx, m m) error
	Delete(tx cursor.Tx, id sql.NamedArg) (sql.Result, error)
}

// Returns a test suite for a CRUD implementation that runs create, retrieve, update,
// delete, and list operations against the given transaction and CRUD implementation.
// A randomization factory is required to create valid models for the tests.
func CRUDTests[M models.Model](tx cursor.Tx, crud CRUD[M], factory Factory[M]) func(t *testing.T) {
	return func(t *testing.T) {
		// NOTE: the tests below are expected to be run in order, do not use t.Parallel()
		require := require.New(t)
		ids := make(IDSet, 0, 16)

		// Create and retreive 16 models
		t.Run("Create&Retrieve", func(t *testing.T) {
			for i := 0; i < 16; i++ {
				m := factory.Make()
				_, err := crud.Create(tx, m)
				require.NoError(err, "should be able to create model")

				id := factory.ID(m)
				ids = ids.Add(id)

				o, err := crud.Retrieve(tx, id)
				require.NoError(err, "should be able to retrieve model")
				require.Equal(m, o, "retrieved model should match created model")
			}

			require.Len(ids, 16, "should have created 16 models with unique IDs")
		})

		// List should return all 16 models without any filtering.
		t.Run("List", func(t *testing.T) {
			cursor, err := crud.List(tx, nil)
			require.NoError(err, "should be able to list models")

			models, err := cursor.List()
			require.NoError(err, "should be able to list models")
			require.Equal(16, len(models), "should return 16 models")

			for _, model := range models {
				require.True(ids.Contains(factory.ID(model)), "should contain model with ID %s", factory.ID(model))
			}
		})

		// Update should update the model with the given ID.
		t.Run("Update", func(t *testing.T) {
			for _, id := range ids {
				model, err := crud.Retrieve(tx, id)
				require.NoError(err, "should be able to retrieve model")

				// TODO: should the factory have an modify method?

				err = crud.Update(tx, model)
				require.NoError(err, "should be able to update model")
			}
		})

		// Delete should remove the models with the given IDs.
		t.Run("Delete", func(t *testing.T) {
			for _, id := range ids {
				_, err := crud.Delete(tx, id)
				require.NoError(err, "should be able to delete model")

				_, err = crud.Retrieve(tx, id)
				require.ErrorIs(err, errors.ErrNotFound, "should not be able to retrieve deleted model")
			}

			cursor, err := crud.List(tx, nil)
			require.NoError(err, "should be able to get a list cursor for the model")

			models, err := cursor.List()
			require.NoError(err, "should be able to create an array of models from the list cursor")
			require.Len(models, 0, "there should be no models in the list because they were deleted")
		})
	}
}

type IDSet []sql.NamedArg

func (s IDSet) Contains(id sql.NamedArg) bool {
	for _, a := range s {
		if a.Value == id.Value {
			return true
		}
	}
	return false
}

func (s IDSet) Add(id sql.NamedArg) IDSet {
	if !s.Contains(id) {
		return append(s, id)
	}
	return s
}
