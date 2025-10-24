package emails

import (
	"net/url"

	"go.rtnl.ai/commo/commo"
	"go.rtnl.ai/x/vero"
)

// ===========================================================================
// Reset Password Email
// ===========================================================================

const (
	ResetPasswordRE       = "Quarterdeck password reset request"
	ResetPasswordTemplate = "reset_password"
)

func NewResetPasswordEmail(recipient string, data ResetPasswordEmailData) (*commo.Email, error) {
	return commo.New(recipient, ResetPasswordRE, ResetPasswordTemplate, data)
}

// ResetPasswordEmailData is used to complete the reset_password template.
type ResetPasswordEmailData struct {
	ContactName  string                 // the user's name, if available
	BaseURL      *url.URL               // the Quarterdeck node's url
	Token        vero.VerificationToken // verification token for reset password link record
	SupportEmail string                 // the Quarterdeck node's support email address
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
