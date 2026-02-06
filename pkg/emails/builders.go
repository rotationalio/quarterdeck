package emails

import (
	"fmt"
	"net/url"

	"go.rtnl.ai/commo"
	"go.rtnl.ai/x/vero"
)

// ===========================================================================
// Email Base Data
// ===========================================================================

// EmailBaseData covers app and org data for email templates.
type EmailBaseData struct {
	AppName        string   // Descriptive name for the application
	AppLogoURL     *url.URL // Application's logo URL
	OrgName        string   // Descriptive name for the organization
	OrgAddress     string   // Organization's street address
	OrgHomepageURL *url.URL // Organization's homepage URL
	SupportEmail   string   // the support email address
}

// ===========================================================================
// Welcome New User Email
// ===========================================================================

// WelcomeUserEmailData is used to complete the welcome_user template.
type WelcomeUserEmailData struct {
	EmailBaseData
	ContactName          string                 // the user's name, if available
	PasswordLinkBaseURL  *url.URL               // the app url
	Token                vero.VerificationToken // verification token for reset password link record
	WelcomeEmailBodyText string                 // the body of the email in text format
	WelcomeEmailBodyHTML string                 // the body of the email in html format
}

func NewWelcomeUserEmail(recipient string, data WelcomeUserEmailData) (*commo.Email, error) {
	// "Join ORGNAME in APPNAME"
	subject := fmt.Sprintf("Join %s in %s", data.OrgName, data.AppName)
	return commo.New(recipient, subject, "welcome_user", data)
}

func (s WelcomeUserEmailData) VerifyURL() string {
	if s.PasswordLinkBaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", s.Token.String())

	s.PasswordLinkBaseURL.RawQuery = params.Encode()
	return s.PasswordLinkBaseURL.String()
}

// ===========================================================================
// Reset Password Email
// ===========================================================================

// ResetPasswordEmailData is used to complete the reset_password template.
type ResetPasswordEmailData struct {
	EmailBaseData
	ContactName         string                 // the user's name, if available
	PasswordLinkBaseURL *url.URL               // the app url
	Token               vero.VerificationToken // verification token for reset password link record
}

func NewResetPasswordEmail(recipient string, data ResetPasswordEmailData) (*commo.Email, error) {
	// "APPNAME password reset request"
	subject := fmt.Sprintf("%s password reset request", data.AppName)
	return commo.New(recipient, subject, "reset_password", data)
}

func (s ResetPasswordEmailData) VerifyURL() string {
	if s.PasswordLinkBaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", s.Token.String())

	s.PasswordLinkBaseURL.RawQuery = params.Encode()
	return s.PasswordLinkBaseURL.String()
}
