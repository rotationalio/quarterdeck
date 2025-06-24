/*
Scene provides well structured template contexts and functionality for HTML template
rendering. We chose the word "scene" to represent the context since "context" is an
overloaded term and milieu was too hard to spell.
*/
package scene

import (
	"github.com/gin-gonic/gin"
	"go.rtnl.ai/quarterdeck/pkg"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/ulid"
)

var (
	// Compute the version of the package at runtime so it is static for all contexts.
	version      = pkg.Version(false)
	shortVersion = pkg.Version(true)
)

// Keys for default Scene context items
const (
	Version         = "Version"
	ShortVersion    = "ShortVersion"
	Page            = "Page"
	IsAuthenticated = "IsAuthenticated"
	User            = "User"
	APIData         = "APIData"
	Parent          = "Parent"
)

type Scene map[string]interface{}

func New(c *gin.Context) Scene {
	if c == nil {
		return Scene{
			Version:      version,
			ShortVersion: shortVersion,
		}
	}

	// Create the basic context
	context := Scene{
		Version:      version,
		ShortVersion: shortVersion,
		Page:         c.Request.URL.Path,
	}

	// Does the user exist in the gin context?
	// if claims, err := auth.GetClaims(c); err != nil {
	// 	context[IsAuthenticated] = false
	// 	context[User] = nil
	// } else {
	// 	context[IsAuthenticated] = true
	// 	context[User] = claims
	// }

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

//===========================================================================
// Scene User Related Helpers
//===========================================================================

func (s Scene) IsAuthenticated() bool {
	if isauths, ok := s[IsAuthenticated]; ok {
		return isauths.(bool)
	}
	return false
}

// func (s Scene) GetUser() *auth.Claims {
// 	if s.IsAuthenticated() {
// 		if claims, ok := s[User]; ok {
// 			if user, ok := claims.(*auth.Claims); ok {
// 				return user
// 			}
// 		}
// 	}
// 	return nil
// }

// func (s Scene) HasRole(role string) bool {
// 	if user := s.GetUser(); user != nil {
// 		return user.Role == role
// 	}
// 	return false
// }

//===========================================================================
// Scene API Data Related Helpers
//===========================================================================

//===========================================================================
// Set Global Scene for Context
//===========================================================================

func WithConf(conf *config.Config) {}
