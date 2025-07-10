package api

import (
	"time"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Next     string `json:"next,omitempty"` // Optional redirect URL after login
}

type LoginReply struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	LastLogin    time.Time `json:"last_login,omitempty"`
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
