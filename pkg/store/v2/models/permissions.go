package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/tidal"
)

type Permission struct {
	ID          int64
	Title       string
	Description string
	Created     time.Time
	Modified    time.Time
}

var _ tidal.Model = (*Permission)(nil)
var _ tidal.Preparer = (*Permission)(nil)

func (p *Permission) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.Create:
		return []string{
			"title",
			"description",
			"created",
			"modified",
		}
	default:
		return []string{
			"id",
			"title",
			"description",
			"created",
			"modified",
		}
	}
}

func (p *Permission) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", p.ID),
			sql.Named("title", p.Title),
			sql.Named("description", p.Description),
			sql.Named("modified", p.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("title", p.Title),
			sql.Named("description", p.Description),
			sql.Named("created", p.Created),
			sql.Named("modified", p.Modified),
		}
	}
}

func (p *Permission) Scan(op tidal.Operation, s tidal.Scanner) error {
	return s.Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.Created,
		&p.Modified,
	)
}

func (p *Permission) Prepare(op tidal.Operation) {
	switch op {
	case tidal.Create:
		p.Created = time.Now().UTC()
		p.Modified = p.Created
	case tidal.Update:
		p.Modified = time.Now().UTC()
	}
}

// PermissionTitles returns the title of each permission in order.
func PermissionTitles(permissions []Permission) []string {
	if len(permissions) == 0 {
		return []string{}
	}
	titles := make([]string, len(permissions))
	for i, permission := range permissions {
		titles[i] = permission.Title
	}
	return titles
}
