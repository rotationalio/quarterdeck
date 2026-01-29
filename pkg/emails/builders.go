package emails

import (
	"fmt"
	"net/url"

	"go.rtnl.ai/commo"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/x/vero"
)

// ===========================================================================
// New User Email
// ===========================================================================

// WelcomeUserEmailData is used to complete the welcome_user template.
type WelcomeUserEmailData struct {
	ContactName  string                 // the user's name, if available
	BaseURL      *url.URL               // the Quarterdeck node's url
	Token        vero.VerificationToken // verification token for reset password link record
	SupportEmail string                 // the Quarterdeck node's support email address
	App          *models.Application    // the Application to send the email from
}

func NewWelcomeUserEmail(recipient string, data WelcomeUserEmailData) (*commo.Email, error) {
	return commo.New(recipient, fmt.Sprintf("Welcome to %s", data.App.DisplayName), data.App.ClientID, data)
}

func (s WelcomeUserEmailData) VerifyURL() string {
	if s.BaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", s.Token.String())

	s.BaseURL.RawQuery = params.Encode()
	return s.BaseURL.String()
}

// ===========================================================================
// Reset Password Email
// ===========================================================================

const (
	ResetPasswordRE       = "Quarterdeck password reset request"
	ResetPasswordTemplate = "reset_password"
)

// ResetPasswordEmailData is used to complete the reset_password template.
type ResetPasswordEmailData struct {
	ContactName  string                 // the user's name, if available
	BaseURL      *url.URL               // the Quarterdeck node's url
	Token        vero.VerificationToken // verification token for reset password link record
	SupportEmail string                 // the Quarterdeck node's support email address
}

func NewResetPasswordEmail(recipient string, data ResetPasswordEmailData) (*commo.Email, error) {
	return commo.New(recipient, ResetPasswordRE, ResetPasswordTemplate, data)
}

func (s ResetPasswordEmailData) VerifyURL() string {
	if s.BaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", s.Token.String())

	s.BaseURL.RawQuery = params.Encode()
	return s.BaseURL.String()
}
