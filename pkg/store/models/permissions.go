package models

import (
	"database/sql"
	"time"
)

type Role struct {
	ID          int64
	Title       string
	Description string
	IsDefault   bool
	Created     time.Time
	Modified    time.Time

	// Associated Fields
	Permissions Permissions
}

type Permission struct {
	ID          int64
	Title       string
	Description string
	Created     time.Time
	Modified    time.Time
}

type Permissions []*Permission

type Roles []*Role

var (
	_ Model = (*Role)(nil)
	_ Model = (*Permission)(nil)
)

var (
	roleFields = [6]string{
		"id",
		"title",
		"description",
		"is_default",
		"created",
		"modified",
	}

	permissionFields = [5]string{
		"id",
		"title",
		"description",
		"created",
		"modified",
	}
)

//===========================================================================
// Scanning and Params
//===========================================================================

// Scanner is an interface for scanning database rows into the Role structs.
func (r *Role) Scan(op Operation, scanner Scanner) error {
	return scanner.Scan(
		&r.ID,
		&r.Title,
		&r.Description,
		&r.IsDefault,
		&r.Created,
		&r.Modified,
	)
}

func (r *Role) Fields(op Operation) []string {
	return roleFields[:]
}

// Params returns all Role fields as named params to be used in a SQL query.
func (r *Role) Params(_ Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("id", r.ID),
		sql.Named("title", r.Title),
		sql.Named("description", r.Description),
		sql.Named("isDefault", r.IsDefault),
		sql.Named("created", r.Created),
		sql.Named("modified", r.Modified),
	}
}

// Scan the Permission struct from a database row.
func (p *Permission) Scan(op Operation, scanner Scanner) error {
	return scanner.Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.Created,
		&p.Modified,
	)
}

func (p *Permission) Fields(op Operation) []string {
	return permissionFields[:]
}

// Params returns all Permission fields as named params to be used in a SQL query.
func (p *Permission) Params(_ Operation) []sql.NamedArg {
	return []sql.NamedArg{
		sql.Named("id", p.ID),
		sql.Named("title", p.Title),
		sql.Named("description", p.Description),
		sql.Named("created", p.Created),
		sql.Named("modified", p.Modified),
	}
}

//============================================================================
// Helper Methods
//============================================================================

func (r Roles) List() []string {
	out := make([]string, 0, len(r))
	for _, role := range r {
		out = append(out, role.Title)
	}
	return out
}

func (p Permissions) List() []string {
	out := make([]string, 0, len(p))
	for _, perm := range p {
		out = append(out, perm.Title)
	}
	return out
}

func (r Roles) Load(in []string) {
	r = make(Roles, 0, len(in))
	for _, title := range in {
		if title != "" {
			r = append(r, &Role{Title: title})
		}
	}
}

func (p Permissions) Load(in []string) {
	p = make(Permissions, 0, len(in))
	for _, title := range in {
		if title != "" {
			p = append(p, &Permission{Title: title})
		}
	}
}
