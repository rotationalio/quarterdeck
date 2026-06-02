package api

import (
	"database/sql"
	"errors"
	"net/mail"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/store/cursor"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

type User struct {
	ID          ulid.ULID `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email"`
	Avatar      string    `json:"avatar,omitempty"`
	LastLogin   time.Time `json:"last_login,omitempty"`
	Roles       []*Role   `json:"roles"`
	Permissions []string  `json:"permissions"`
	Created     time.Time `json:"created,omitempty"`
	Modified    time.Time `json:"modified,omitempty"`
}

type UserList struct {
	Page  *UserPage
	Users []*User
}

type UserPage struct {
	Page
	Role string `json:"role,omitempty"`
}

type UserPageQuery struct {
	PageQuery
	Role string `json:"role,omitempty"`
}

func NewUser(model *models.User) (out *User, err error) {
	out = &User{
		ID:          model.ID,
		Email:       model.Email,
		Avatar:      model.Gravatar(),
		Permissions: model.Permissions.List(),
		Created:     model.Created,
		Modified:    model.Modified,
	}

	out.Roles = make([]*Role, 0, len(model.Roles))
	for _, role := range model.Roles {
		out.Roles = append(out.Roles, &Role{
			ID:    int(role.ID),
			Title: role.Title,
		})
	}

	if model.Name.Valid {
		out.Name = model.Name.String
	}

	if model.LastLogin.Valid {
		out.LastLogin = model.LastLogin.Time
	}

	return out, nil
}

func NewUserList(list cursor.Cursor[*models.User]) (out *UserList, err error) {
	return nil, errors.New("not implemented")
}

func (u *User) Validate() (err error) {
	if !u.ID.IsZero() {
		err = ValidationError(err, ReadOnlyField("id"))
	}

	if u.Email == "" {
		err = ValidationError(err, MissingField("email"))
	} else {
		if _, perr := mail.ParseAddress(u.Email); perr != nil {
			err = ValidationError(err, IncorrectField("email", perr.Error()))
		}
	}

	if !u.LastLogin.IsZero() {
		err = ValidationError(err, ReadOnlyField("last_login"))
	}

	// TODO validate Roles

	if len(u.Permissions) != 0 {
		err = ValidationError(err, ReadOnlyField("permissions"))
	}

	if !u.Created.IsZero() {
		err = ValidationError(err, ReadOnlyField("created"))
	}

	if !u.Modified.IsZero() {
		err = ValidationError(err, ReadOnlyField("modified"))
	}

	return err
}

func (u *User) Model() (model *models.User, err error) {
	model = &models.User{
		BaseModel: models.BaseModel{ID: u.ID},
		Name:      sql.NullString{Valid: u.Name != "", String: u.Name},
		Email:     u.Email,
		Roles:     make(models.Roles, 0, len(u.Roles)),
	}

	for _, role := range u.Roles {
		model.Roles = append(model.Roles, role.Model())
	}

	return model, nil
}

func (p *UserPageQuery) UserPage() (page *UserPage) {
	page = &UserPage{
		Page: Page{
			NextPageToken: p.NextPageToken,
			PageSize:      p.PageSize,
		},
		Role: p.Role,
	}

	return page
}
