package api

import (
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
