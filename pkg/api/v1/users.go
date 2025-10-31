package api

import (
	"database/sql"
	"net/mail"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

type User struct {
	ID          ulid.ULID `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email"`
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
		Permissions: model.Permissions(),
		Created:     model.Created,
		Modified:    model.Modified,
	}

	var roles []*models.Role
	if roles, err = model.Roles(); err != nil {
		return nil, err
	}
	out.Roles = make([]*Role, 0, len(roles))
	for _, role := range roles {
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

func NewUserList(list *models.UserList) (out *UserList, err error) {
	out = &UserList{
		Users: make([]*User, 0, len(list.Users)),
	}

	if out.Page, err = NewUserPage(list.Page); err != nil {
		return nil, err
	}

	for _, modelUser := range list.Users {
		var user *User
		if user, err = NewUser(modelUser); err != nil {
			return nil, err
		}
		out.Users = append(out.Users, user)
	}

	return out, nil
}

func NewUserPage(page *models.UserPage) (out *UserPage, err error) {
	out = &UserPage{
		Page: Page{
			NextPageToken: page.NextPageID.String(),
			PrevPageToken: page.PrevPageID.String(),
			PageSize:      int(page.PageSize),
		},
		Role: page.Role,
	}

	return out, nil
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
		Model: models.Model{ID: u.ID},
		Name:  sql.NullString{Valid: u.Name != "", String: u.Name},
		Email: u.Email,
	}

	modelRoles := make([]*models.Role, 0, len(u.Roles))
	for _, role := range u.Roles {
		modelRoles = append(modelRoles, role.Model())
	}
	model.SetRoles(modelRoles)

	return model, nil
}

func (p *UserPage) Model() (model *models.UserPage, err error) {
	var next, prev ulid.ULID

	if next, err = ulid.Parse(p.NextPageToken); err != nil {
		return nil, err
	}

	if prev, err = ulid.Parse(p.PrevPageToken); err != nil {
		return nil, err
	}

	model = &models.UserPage{
		Page: models.Page{
			NextPageID: next,
			PrevPageID: prev,
			PageSize:   uint32(p.PageSize),
		},
		Role: p.Role,
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
