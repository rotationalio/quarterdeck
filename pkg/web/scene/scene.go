/*
Scene provides well structured template contexts and functionality for HTML template
rendering. We chose the word "scene" to represent the context since "context" is an
overloaded term and milieu was too hard to spell.
*/
package scene

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/api/v1"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/ulid"
)

var (
	// Compute the version of the package at runtime so it is static for all contexts.
	version      = pkg.Version(false)
	shortVersion = pkg.Version(true)
	revision     = pkg.GitVersion
	buildDate    = pkg.BuildDate

	// Authentication URLs
	issuer            *url.URL
	loginURL          string
	forgotPasswordURL string
)

// Keys for default Scene context items
const (
	Version         = "Version"
	ShortVersion    = "ShortVersion"
	Revision        = "Revision"
	BuildDate       = "BuildDate"
	Page            = "Page"
	IsAuthenticated = "IsAuthenticated"
	User            = "User"
	UserID          = "UserID"
	APIData         = "APIData"
	Parent          = "Parent"
)

type Scene map[string]interface{}

func New(c *gin.Context) Scene {
	if c == nil {
		return Scene{
			Version:      version,
			ShortVersion: shortVersion,
			Revision:     revision,
			BuildDate:    buildDate,
		}
	}

	// Create the basic context
	context := Scene{
		Version:      version,
		ShortVersion: shortVersion,
		Revision:     revision,
		BuildDate:    buildDate,
		Page:         c.Request.URL.Path,
	}

	// Does the user exist in the gin context?
	if claims, err := auth.GetClaims(c); err != nil {
		context[IsAuthenticated] = false
		context[User] = nil
		context[UserID] = nil
	} else {
		context[IsAuthenticated] = true
		context[User] = claims
		if _, userID, err := claims.SubjectID(); err == nil {
			context[UserID] = userID.String()
		}
	}

	return context
}

func (s Scene) Update(o Scene) Scene {
	for key, val := range o {
		s[key] = val
	}
	return s
}

func (s Scene) WithAPIData(data interface{}) Scene {
	s[APIData] = data
	return s
}

func (s Scene) WithParent(parent ulid.ULID) Scene {
	s[Parent] = parent.String()
	return s
}

func (s Scene) With(key string, val interface{}) Scene {
	s[key] = val
	return s
}

func (s Scene) ForPage(page string) Scene {
	s[Page] = page
	return s
}

//===========================================================================
// Scene User Related Helpers
//===========================================================================

func (s Scene) IsAuthenticated() bool {
	if isauths, ok := s[IsAuthenticated]; ok {
		return isauths.(bool)
	}
	return false
}

func (s Scene) GetUser() *auth.Claims {
	if s.IsAuthenticated() {
		if claims, ok := s[User]; ok {
			if user, ok := claims.(*auth.Claims); ok {
				return user
			}
		}
	}
	return nil
}

func (s Scene) HasPermission(permission string) bool {
	if claims := s.GetUser(); claims != nil {
		return claims.HasPermission(permission)
	}
	return false
}

//===========================================================================
// Scene API Data Related Helpers
//===========================================================================

func (s Scene) APIKeysList() *api.APIKeyList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.APIKeyList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) APIKey() *api.APIKey {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.APIKey); ok {
			return out
		}
	}
	return nil
}

//===========================================================================
// Set Global Scene for Context
//===========================================================================

func WithConf(conf config.Config) {
	issuer, _ = url.Parse(conf.Auth.Issuer)
	loginURL = issuer.ResolveReference(&url.URL{Path: "/v1/login"}).String()
	forgotPasswordURL = issuer.ResolveReference(&url.URL{Path: "/forgot-password"}).String()
}
