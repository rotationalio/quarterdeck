package api

import (
	"strings"
	"time"
)

type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	AutoLogout bool   `json:"auto_logout,omitempty"` // Optional flag to omit refresh token and persistent session
	Next       string `json:"next,omitempty"`        // Optional redirect URL after login
}

type LoginReply struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	LastLogin    time.Time `json:"last_login,omitempty"`
}

type AuthenticateRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Next         string `json:"next,omitempty"` // Optional redirect URL after authentication
}

type ReauthenticateRequest struct {
	RefreshToken string `json:"refresh_token"`
	Next         string `json:"next,omitempty"` // Optional redirect URL after re-authentication
}

func (r *LoginRequest) Validate() (err error) {
	if r.Email == "" {
		err = ValidationError(err, MissingField("email"))
	}

	if r.Password == "" {
		err = ValidationError(err, MissingField("password"))
	} else if len(r.Password) < 8 {
		err = ValidationError(err, IncorrectField("password", "must be at least 8 characters long"))
	}

	return err
}

// Redirect returns the next URL to redirect to after a successful login.
// If no next URL is provided, it defaults to the root path ("/").
// TODO: should we allow this to be configurable?
func (r *LoginRequest) Redirect() string {
	// Check if we have a next URL to redirect to with the request.
	if r.Next != "" {
		return r.Next
	}

	// If all else fails, redirect to the root path.
	return "/"
}

func (r *AuthenticateRequest) Validate() (err error) {
	r.ClientID = strings.TrimSpace(r.ClientID)
	if r.ClientID == "" {
		err = ValidationError(err, MissingField("client_id"))
	}

	r.ClientSecret = strings.TrimSpace(r.ClientSecret)
	if r.ClientSecret == "" {
		err = ValidationError(err, MissingField("client_secret"))
	}
	return err
}

func (r *AuthenticateRequest) Redirect() string {
	// Check if we have a next URL to redirect to with the request.
	if r.Next != "" {
		return r.Next
	}

	// If all else fails, redirect to the root path.
	return "/"
}

func (r *ReauthenticateRequest) Validate() (err error) {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	if r.RefreshToken == "" {
		err = ValidationError(err, MissingField("refresh_token"))
	}
	return err
}

func (r *ReauthenticateRequest) Redirect() string {
	// Check if we have a next URL to redirect to with the request.
	if r.Next != "" {
		return r.Next
	}

	// If all else fails, redirect to the root path.
	return "/"
}
