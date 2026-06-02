package api

import (
	"database/sql"
	"errors"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/store/cursor"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

type APIKey struct {
	ID          ulid.ULID  `json:"id,omitempty"`
	Description string     `json:"description"`
	ClientID    string     `json:"client_id"`
	Secret      string     `json:"secret,omitempty"`
	CreatedBy   ulid.ULID  `json:"created_by,omitempty"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
	Permissions []string   `json:"permissions"`
	Created     time.Time  `json:"created,omitempty"`
	Modified    time.Time  `json:"modified,omitempty"`
}

type APIKeyList struct {
	Page    *Page     `json:"page"`
	APIKeys []*APIKey `json:"apikeys"`
}

func NewAPIKey(model *models.APIKey) (out *APIKey, err error) {
	out = &APIKey{
		ID:          model.ID,
		Description: model.Description.String,
		ClientID:    model.ClientID,
		CreatedBy:   model.CreatedBy,
		Permissions: model.Permissions.List(),
		Created:     model.Created,
		Modified:    model.Modified,
	}

	if model.LastSeen.Valid {
		out.LastSeen = &model.LastSeen.Time
	}

	return out, nil
}

func NewAPIKeyList(cursor cursor.Cursor[*models.APIKey]) (out *APIKeyList, err error) {
	return nil, errors.New("not implemented")
}

func (k *APIKey) Validate() (err error) {
	if !k.ID.IsZero() {
		err = ValidationError(err, ReadOnlyField("id"))
	}

	if k.Description == "" {
		err = ValidationError(err, MissingField("description"))
	}

	if k.ClientID != "" {
		err = ValidationError(err, ReadOnlyField("client_id"))
	}

	if k.Secret != "" {
		err = ValidationError(err, ReadOnlyField("secret"))
	}

	if k.LastSeen != nil {
		err = ValidationError(err, ReadOnlyField("last_seen"))
	}

	if !k.CreatedBy.IsZero() {
		err = ValidationError(err, ReadOnlyField("created_by"))
	}

	if !k.Created.IsZero() {
		err = ValidationError(err, ReadOnlyField("created"))
	}

	if !k.Modified.IsZero() {
		err = ValidationError(err, ReadOnlyField("modified"))
	}

	// TODO: validate permissions
	return err
}

func (k *APIKey) Model() (model *models.APIKey, err error) {
	model = &models.APIKey{
		BaseModel: models.BaseModel{
			ID:       k.ID,
			Created:  k.Created,
			Modified: k.Modified,
		},
		Description: sql.NullString{String: k.Description, Valid: k.Description != ""},
		ClientID:    k.ClientID,
		CreatedBy:   k.CreatedBy,
	}

	if k.LastSeen != nil {
		model.LastSeen = sql.NullTime{Time: *k.LastSeen, Valid: true}
	}

	if len(k.Permissions) > 0 {
		model.Permissions.Load(k.Permissions)
	}

	return model, nil
}
