package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/ulid"
)

// BaseModel is embedded into most models to provide ID management and timestamps.
type BaseModel struct {
	ID       ulid.ULID
	Created  time.Time
	Modified time.Time
}

// The interface required for all Quarterdeck models.
type Model interface {
	// Used during select operations to scan the model from a database row.
	Scan(Operation, Scanner) error

	// Used during select operations to identify the fields that should be returned and
	// the order in which they should be returned. Note that during create and update
	// operations, the fields are selected from the sql.NamedArgs returned by the
	// Params method.
	Fields(Operation) []string

	// Used during insert and update operations to supply fields and their values to
	// the database. Note that the names of the parameters must match the fields used
	// in the database schema.
	Params(Operation) []sql.NamedArg
}

// Scanner is an interface for *sql.Rows and *sql.Row so that models can implement how
// they scan fields into their struct without having to specify every field every time.
type Scanner interface {
	Scan(dest ...any) error
}

// Prepare is an interface for models that need to prepare their fields before being
// created or updated in the database. This is called automatically by the store before
// creating or updating a model.
type Preparer interface {
	Prepare(Operation)
}

// Validator is an interface for models that need to have non-database constraints
// validated before being created or updated -- these business rules are not enforced
// by the database, but may be mirrored by database constratints depending on the store
// type.
type Validator interface {
	Validate(Operation) error
}

//===========================================================================
// Base Model Methods
//===========================================================================

var _ Preparer = (*BaseModel)(nil)

// Updates the modified timestamp for the model. If creating a new record, also creates
// a new ULID for the ID field and the created timestamp. Database stores may override
// the values set in this method, see the specific store type for details.
func (b *BaseModel) Prepare(op Operation) {
	switch op {
	case Create:
		b.ID = ulid.MakeSecure()
		b.Created = time.Now().UTC()
		b.Modified = b.Created

	case Update:
		b.Modified = time.Now().UTC()
	}
}

func (b *BaseModel) IsZero() bool {
	return b == nil || (b.ID.IsZero() && b.Created.IsZero() && b.Modified.IsZero())
}

//===========================================================================
// Pagination
//===========================================================================

const DefaultPageSize = uint32(50)

// Page is a struct that contains list information for paginated results or for lists
// that are filtered by specific fields. This struct is both returned after a query to
// describe the list contents and can be used as a query to return a new page of results.
type Page struct {
	PageSize   uint32    `json:"page_size"`
	NextPageID ulid.ULID `json:"next_page_id"`
	PrevPageID ulid.ULID `json:"prev_page_id"`
}

func PageFrom(in *Page) (out *Page) {
	out = &Page{
		PageSize: DefaultPageSize,
	}

	if in != nil && in.PageSize > 0 {
		out.PageSize = in.PageSize
	}

	return out
}

//============================================================================
// Operations
//============================================================================

type Operation uint8

const (
	Unknown Operation = iota
	List
	Create
	Retrieve
	Update
	Delete
)

var (
	SelectOperations  = [2]Operation{List, Retrieve}
	EditOperations    = [3]Operation{Create, Update, Delete}
	PrepareOperations = [2]Operation{Create, Update}
)

func (o Operation) String() string {
	switch o {
	case List:
		return "List"
	case Create:
		return "Create"
	case Retrieve:
		return "Retrieve"
	case Update:
		return "Update"
	case Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}
