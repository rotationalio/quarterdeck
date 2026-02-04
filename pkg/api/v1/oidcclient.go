package api

import (
	"database/sql"
	"fmt"
	"net/mail"
	"net/url"
	"time"

	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/ulid"
)

type OIDCClient struct {
	ID           ulid.ULID  `json:"id,omitempty"`
	ClientName   string     `json:"client_name"`
	ClientURI    *string    `json:"client_uri,omitempty"`
	LogoURI      *string    `json:"logo_uri,omitempty"`
	PolicyURI    *string    `json:"policy_uri,omitempty"`
	TOSURI       *string    `json:"tos_uri,omitempty"`
	Contacts     []string   `json:"contacts,omitempty"`
	RedirectURIs []string   `json:"redirect_uris"`
	ClientID     string     `json:"client_id,omitempty"`
	Secret       string     `json:"secret,omitempty"`
	CreatedBy    ulid.ULID  `json:"created_by,omitempty"`
	Created      time.Time  `json:"created,omitempty"`
	Modified     time.Time  `json:"modified,omitempty"`
}

type OIDCClientList struct {
	Page        *Page         `json:"page"`
	OIDCClients []*OIDCClient `json:"oidc_clients"`
}

// NewOIDCClient converts a store model to an API DTO. Secret is never set on
// the returned client.
func NewOIDCClient(model *models.OIDCClient) (out *OIDCClient, err error) {
	out = &OIDCClient{
		ID:           model.ID,
		ClientName:   model.ClientName,
		RedirectURIs: model.RedirectURIs,
		ClientID:     model.ClientID,
		CreatedBy:    model.CreatedBy,
		Created:      model.Created,
		Modified:     model.Modified,
	}

	if model.ClientURI.Valid && model.ClientURI.String != "" {
		s := model.ClientURI.String
		out.ClientURI = &s
	}
	if model.LogoURI.Valid && model.LogoURI.String != "" {
		s := model.LogoURI.String
		out.LogoURI = &s
	}
	if model.PolicyURI.Valid && model.PolicyURI.String != "" {
		s := model.PolicyURI.String
		out.PolicyURI = &s
	}
	if model.TOSURI.Valid && model.TOSURI.String != "" {
		s := model.TOSURI.String
		out.TOSURI = &s
	}
	if len(model.Contacts) > 0 {
		out.Contacts = make([]string, 0, len(model.Contacts))
		for _, c := range model.Contacts {
			if c.Valid {
				out.Contacts = append(out.Contacts, c.String)
			}
		}
	}

	return out, nil
}

// NewOIDCClientList converts a store model list to an API list.
func NewOIDCClientList(list *models.OIDCClientList) (out *OIDCClientList, err error) {
	out = &OIDCClientList{
		Page:        &Page{},
		OIDCClients: make([]*OIDCClient, 0, len(list.OIDCClients)),
	}

	if list.Page != nil {
		out.Page.PrevPageToken = list.Page.PrevPageID.String()
		out.Page.NextPageToken = list.Page.NextPageID.String()
		out.Page.PageSize = int(list.Page.PageSize)
	}

	for _, model := range list.OIDCClients {
		var client *OIDCClient
		if client, err = NewOIDCClient(model); err != nil {
			return nil, err
		}
		out.OIDCClients = append(out.OIDCClients, client)
	}

	return out, nil
}

// Validate validates the OIDCClient. If create is true, the ID field is not
// allowed to be set.
func (o *OIDCClient) Validate(create bool) (err error) {
	// check if ID is set on create
	if create && !o.ID.IsZero() {
		err = ValidationError(err, ReadOnlyField("id"))
	}

	// readonly fields
	if o.ClientID != "" {
		err = ValidationError(err, ReadOnlyField("client_id"))
	}

	if o.Secret != "" {
		err = ValidationError(err, ReadOnlyField("secret"))
	}

	if !o.CreatedBy.IsZero() {
		err = ValidationError(err, ReadOnlyField("created_by"))
	}

	if !o.Created.IsZero() {
		err = ValidationError(err, ReadOnlyField("created"))
	}

	if !o.Modified.IsZero() {
		err = ValidationError(err, ReadOnlyField("modified"))
	}

	// redirect_uris: at least one required; each must be valid URL
	if len(o.RedirectURIs) == 0 {
		err = ValidationError(err, MissingField("redirect_uris"))
	} else {
		for i, u := range o.RedirectURIs {
			field := fmt.Sprintf("redirect_uris[%d]", i)
			if perr := validateURI(field, u); perr != nil {
				err = ValidationError(err, IncorrectField(field, perr.Error()))
			}
		}
	}

	// Optional URIs: when present, must be valid absolute URLs
	if o.ClientURI != nil && *o.ClientURI != "" {
		if perr := validateURI("client_uri", *o.ClientURI); perr != nil {
			err = ValidationError(err, IncorrectField("client_uri", perr.Error()))
		}
	}

	if o.LogoURI != nil && *o.LogoURI != "" {
		if perr := validateURI("logo_uri", *o.LogoURI); perr != nil {
			err = ValidationError(err, IncorrectField("logo_uri", perr.Error()))
		}
	}

	if o.PolicyURI != nil && *o.PolicyURI != "" {
		if perr := validateURI("policy_uri", *o.PolicyURI); perr != nil {
			err = ValidationError(err, IncorrectField("policy_uri", perr.Error()))
		}
	}

	if o.TOSURI != nil && *o.TOSURI != "" {
		if perr := validateURI("tos_uri", *o.TOSURI); perr != nil {
			err = ValidationError(err, IncorrectField("tos_uri", perr.Error()))
		}
	}

	// contacts: optional; when present, each must be valid email
	for i, c := range o.Contacts {
		if c == "" {
			continue
		}
		if _, perr := mail.ParseAddress(c); perr != nil {
			err = ValidationError(err, IncorrectField(fmt.Sprintf("contacts[%d]", i), perr.Error()))
		}
	}

	return err
}

func validateURI(field, raw string) error {
	if raw == "" {
		return fmt.Errorf("%s: cannot be empty", field)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if !parsed.IsAbs() || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s: must be an absolute URL with scheme and host", field)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s: scheme must be http or https", field)
	}
	return nil
}

// Model converts the API DTO to a store model for create/update. ID must be set by caller when updating.
func (o *OIDCClient) Model() (model *models.OIDCClient, err error) {
	model = &models.OIDCClient{
		Model:        models.Model{ID: o.ID, Created: o.Created, Modified: o.Modified},
		ClientName:   o.ClientName,
		RedirectURIs: o.RedirectURIs,
		ClientID:     o.ClientID,
		Secret:       o.Secret,
		CreatedBy:    o.CreatedBy,
	}

	if o.ClientURI != nil && *o.ClientURI != "" {
		model.ClientURI = sql.NullString{String: *o.ClientURI, Valid: true}
	}
	if o.LogoURI != nil && *o.LogoURI != "" {
		model.LogoURI = sql.NullString{String: *o.LogoURI, Valid: true}
	}
	if o.PolicyURI != nil && *o.PolicyURI != "" {
		model.PolicyURI = sql.NullString{String: *o.PolicyURI, Valid: true}
	}
	if o.TOSURI != nil && *o.TOSURI != "" {
		model.TOSURI = sql.NullString{String: *o.TOSURI, Valid: true}
	}
	if len(o.Contacts) > 0 {
		model.Contacts = make([]sql.NullString, len(o.Contacts))
		for i, c := range o.Contacts {
			model.Contacts[i] = sql.NullString{String: c, Valid: c != ""}
		}
	}

	return model, nil
}
