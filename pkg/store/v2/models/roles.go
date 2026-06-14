package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/tidal"
)

type Role struct {
	ID          int64
	Title       string
	Description string
	IsDefault   bool
	Created     time.Time
	Modified    time.Time
	Permissions []Permission
}

var _ tidal.Model = (*Role)(nil)
var _ tidal.Preparer = (*Role)(nil)

func (r *Role) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.Create:
		return []string{
			"title",
			"description",
			"is_default",
			"created",
			"modified",
		}
	default:
		return []string{
			"id",
			"title",
			"description",
			"is_default",
			"created",
			"modified",
		}
	}
}

func (r *Role) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", r.ID),
			sql.Named("title", r.Title),
			sql.Named("description", r.Description),
			sql.Named("is_default", r.IsDefault),
			sql.Named("modified", r.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("title", r.Title),
			sql.Named("description", r.Description),
			sql.Named("is_default", r.IsDefault),
			sql.Named("created", r.Created),
			sql.Named("modified", r.Modified),
		}
	}
}

func (r *Role) Scan(op tidal.Operation, s tidal.Scanner) error {
	return s.Scan(
		&r.ID,
		&r.Title,
		&r.Description,
		&r.IsDefault,
		&r.Created,
		&r.Modified,
	)
}

func (r *Role) Prepare(op tidal.Operation) {
	switch op {
	case tidal.Create:
		r.Created = time.Now().UTC()
		r.Modified = r.Created
	case tidal.Update:
		r.Modified = time.Now().UTC()
	}
}
