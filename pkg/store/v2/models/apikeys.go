package models

import (
	"database/sql"
	"time"

	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/enum"
	qerrors "go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/ulid"
)

// API Keys are considered stale once they have not been used in this duration.
const APIKeyStalenessThreshold = 90 * 24 * time.Hour

type APIKey struct {
	tidal.BaseModel
	Description sql.NullString
	ClientID    string
	Secret      string
	CreatedBy   ulid.ULID
	LastSeen    sql.NullTime
	Revoked     sql.NullTime
	Permissions []Permission
}

var _ tidal.Model = (*APIKey)(nil)
var _ tidal.Validator = (*APIKey)(nil)

func (k *APIKey) Fields(op tidal.Operation) []string {
	switch op {
	case tidal.List:
		return []string{
			"id",
			"description",
			"client_id",
			"created_by",
			"last_seen",
			"revoked",
			"created",
			"modified",
		}
	case tidal.Update:
		return []string{
			"id",
			"description",
			"modified",
		}
	default:
		return []string{
			"id",
			"description",
			"client_id",
			"secret",
			"created_by",
			"last_seen",
			"revoked",
			"created",
			"modified",
		}
	}
}

func (k *APIKey) Params(op tidal.Operation) []sql.NamedArg {
	switch op {
	case tidal.Update:
		return []sql.NamedArg{
			sql.Named("id", k.ID),
			sql.Named("description", k.Description),
			sql.Named("modified", k.Modified),
		}
	default:
		return []sql.NamedArg{
			sql.Named("id", k.ID),
			sql.Named("description", k.Description),
			sql.Named("client_id", k.ClientID),
			sql.Named("secret", k.Secret),
			sql.Named("created_by", k.CreatedBy),
			sql.Named("last_seen", k.LastSeen),
			sql.Named("revoked", k.Revoked),
			sql.Named("created", k.Created),
			sql.Named("modified", k.Modified),
		}
	}
}

func (k *APIKey) Scan(op tidal.Operation, s tidal.Scanner) error {
	switch op {
	case tidal.List:
		return s.Scan(
			&k.ID,
			&k.Description,
			&k.ClientID,
			&k.CreatedBy,
			&k.LastSeen,
			&k.Revoked,
			&k.Created,
			&k.Modified,
		)
	default:
		return s.Scan(
			&k.ID,
			&k.Description,
			&k.ClientID,
			&k.Secret,
			&k.CreatedBy,
			&k.LastSeen,
			&k.Revoked,
			&k.Created,
			&k.Modified,
		)
	}
}

// Validates that ClientID, Secret, and CreatedBy are set on create; default
// [tidal.BaseModel.Validate] runs first.
func (k *APIKey) Validate(op tidal.Operation) error {
	if err := k.BaseModel.Validate(op); err != nil {
		return err
	}
	if op == tidal.Create {
		if k.ClientID == "" || k.Secret == "" || k.CreatedBy.IsZero() {
			return qerrors.ErrZeroValuedNotNull
		}
	}
	return nil
}

// Determines the status of the APIKey:
//
//   - If the API key is revoked, returns [enum.APIKeyStatusRevoked].
//   - If the API key has never been used (LastSeen is unset), returns
//     [enum.APIKeyStatusUnused].
//   - If the API key has not been used in the last [APIKeyStalenessThreshold],
//     returns [enum.APIKeyStatusStale].
//   - Otherwise, returns [enum.APIKeyStatusActive].
func (k *APIKey) Status() enum.APIKeyStatus {
	if k.Revoked.Valid || !k.Revoked.Time.IsZero() {
		return enum.APIKeyStatusRevoked
	}

	if !k.LastSeen.Valid || k.LastSeen.Time.IsZero() {
		return enum.APIKeyStatusUnused
	}

	if time.Since(k.LastSeen.Time) > APIKeyStalenessThreshold {
		return enum.APIKeyStatusStale
	}

	return enum.APIKeyStatusActive
}

func (k APIKey) Claims() *auth.Claims {
	claims := &auth.Claims{
		ClientID:    k.ClientID,
		Permissions: PermissionTitles(k.Permissions),
	}

	claims.SetSubjectID(auth.SubjectAPIKey, k.ID)
	return claims
}
