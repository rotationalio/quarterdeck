package permissions

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Permission uint8

type Permissions []Permission

// These permissions are used to authorize access to various Quarterdeck resources and
// their names should match the permissions defined in the Quarterdeck database.
//
// NOTE: if adding or removing permissions from this list, ensure they are updated in
// a database migration as well. Also ensure the AllPermissions array is also updated.
const (
	Unknown         Permission = iota
	UsersView                  // View users and their details (including invited users)
	UsersManage                // Invite new users, update user details, reset passwords, and delete users
	APIKeysView                // View the API keys created in the system
	APIKeysManage              // Create and update API keys and their permissions
	APIKeysRevoke              // Revoke API keys
	RolesView                  // View the available roles in the system
	PermissionsView            // View the available permissions in the system
	ConfigView                 // View the configuration settings
	ConfigManage               // Update the configuration settings
)

var AllPermissions = [10]Permission{
	UsersView, UsersManage,
	APIKeysView, APIKeysManage, APIKeysRevoke,
	RolesView,
	PermissionsView,
	ConfigView, ConfigManage,
}

var names = [11]string{
	"unknown",
	"users:view", "users:manage",
	"apikeys:view", "apikeys:manage", "apikeys:revoke",
	"roles:view",
	"permissions:view",
	"config:view", "config:manage",
}

func Parse(p any) (Permission, error) {
	switch v := p.(type) {
	case uint8:
		return Permission(v), nil
	case int64:
		return Permission(v), nil
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		for i, name := range names {
			if v == name {
				return Permission(i), nil
			}
		}
		return Unknown, fmt.Errorf("%q is not a valid permission name", v)
	case Permission:
		return v, nil
	default:
		return Unknown, fmt.Errorf("cannot parse type %T as a Permission", p)
	}
}

func (p Permission) String() string {
	if idx := int(p); idx >= 0 && idx < len(names) {
		return names[idx]
	}
	return names[0]
}

func (p Permission) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *Permission) UnmarshalJSON(data []byte) (err error) {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}

	if *p, err = Parse(name); err != nil {
		return err
	}
	return nil
}

func (p Permissions) String() []string {
	names := make([]string, len(p))
	for i, perm := range p {
		names[i] = perm.String()
	}
	return names
}
