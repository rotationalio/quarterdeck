package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.rtnl.ai/ulid"
)

// Quarterdeck claims are serialized into a JWT token and contain authentication (via
// the signed token), authorization (via the role and permissions), and client
// information such as the client ID for API keys or user profile information.
type Claims struct {
	jwt.RegisteredClaims
	ClientID    string   `json:"clientID,omitempty"`    // Only used for API keys, not users.
	Name        string   `json:"name,omitempty"`        // Only used for users, not API keys.
	Email       string   `json:"email,omitempty"`       // Only used for users, not API keys.
	Gravatar    string   `json:"gravatar,omitempty"`    // Only used for users, not API keys.
	Role        string   `json:"role,omitempty"`        // The role assigned to a user (not used with API keys).
	Permissions []string `json:"permissions,omitempty"` // The permissions assigned to the claims.
}

//===========================================================================
// Claims Methods
//===========================================================================

func (c *Claims) SetSubjectID(sub SubjectType, id ulid.ULID) {
	c.Subject = fmt.Sprintf("%c%s", sub, id)
}

func (c Claims) SubjectID() (SubjectType, ulid.ULID, error) {
	sub := SubjectType(c.Subject[0])
	id, err := ulid.Parse(c.Subject[1:])
	return sub, id, err
}

func (c Claims) SubjectType() SubjectType {
	return SubjectType(c.Subject[0])
}

func (c Claims) HasPermission(required string) bool {
	for _, permisison := range c.Permissions {
		if permisison == required {
			return true
		}
	}
	return false
}

func (c Claims) HasAllPermissions(required ...string) bool {
	if len(required) == 0 {
		return false
	}

	for _, perm := range required {
		if !c.HasPermission(perm) {
			return false
		}
	}
	return true
}

//===========================================================================
// JWT Unverified Timestamp Extraction
//===========================================================================

// Used to extract expiration and not before timestamps without having to use public keys
var tsparser = jwt.NewParser(jwt.WithoutClaimsValidation())

func ParseUnverified(tks string) (claims *jwt.RegisteredClaims, err error) {
	claims = &jwt.RegisteredClaims{}
	if _, _, err = tsparser.ParseUnverified(tks, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func ExpiresAt(tks string) (_ time.Time, err error) {
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(tks); err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}

func NotBefore(tks string) (_ time.Time, err error) {
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(tks); err != nil {
		return time.Time{}, err
	}
	return claims.NotBefore.Time, nil
}
