package models

import (
	"time"

	"go.rtnl.ai/ulid"
)

// Model is the base for all models stored in the database.
type Model struct {
	ID       ulid.ULID `json:"id"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

// Scanner is an interface for *sql.Rows and *sql.Row so that models can implement how
// they scan fields into their struct without having to specify every field every time.
type Scanner interface {
	Scan(dest ...any) error
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
