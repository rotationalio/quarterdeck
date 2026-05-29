package emails

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	texttemplate "text/template"

	"go.rtnl.ai/commo"
	"go.rtnl.ai/quarterdeck/pkg/store/models"
	"go.rtnl.ai/x/vero"
)

// ============================================================================
// Email base data
// ============================================================================

// EmailBaseData covers app and org data for email templates.
type EmailBaseData struct {
	AppName        string   // Descriptive name for the application
	AppLogoURL     *url.URL // Application's logo URL
	OrgName        string   // Descriptive name for the organization
	OrgHomepageURL *url.URL // Organization's homepage URL
	SupportEmail   string   // the support email address
}

// ============================================================================
// Welcome user email
// ============================================================================

// WelcomeUserEmailData is used to complete the welcome_user template.
type WelcomeUserEmailData struct {
	EmailBaseData
	ContactName          string                 // the user's name, if available
	Role                 string                 // role title for custom welcome body templates
	PasswordResetURL     *url.URL               // the app url
	Token                vero.VerificationToken // verification token for reset password link record
	WelcomeEmailBodyText string                 // the body of the email in text format
	WelcomeEmailBodyHTML template.HTML          // the body of the email in html format
}

// RoleTitle returns the first role title for the user, or empty if none.
func RoleTitle(user *models.User) string {
	roles, err := user.Roles()
	if err != nil || len(roles) == 0 {
		return ""
	}
	return roles[0].Title
}

// VerifyURL returns the password-reset URL including the signed verification token.
func (d WelcomeUserEmailData) VerifyURL() string {
	if d.PasswordResetURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", d.Token.String())

	d.PasswordResetURL.RawQuery = params.Encode()
	return d.PasswordResetURL.String()
}

// RenderedWelcomeBodyText evaluates WelcomeEmailBodyText as nested text/template.
func (d WelcomeUserEmailData) RenderedWelcomeBodyText() (string, error) {
	if d.WelcomeEmailBodyText == "" {
		return "", nil
	}
	t, err := texttemplate.New("welcome_body").Parse(d.WelcomeEmailBodyText)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderedWelcomeBodyHTML evaluates WelcomeEmailBodyHTML as nested html/template with the same data.
func (d WelcomeUserEmailData) RenderedWelcomeBodyHTML() (template.HTML, error) {
	if len(d.WelcomeEmailBodyHTML) == 0 {
		return "", nil
	}
	t, err := template.New("welcome_body").Parse(string(d.WelcomeEmailBodyHTML))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// NewWelcomeUserEmail builds a welcome_user commo email for the recipient.
func NewWelcomeUserEmail(recipient string, data WelcomeUserEmailData) (*commo.Email, error) {
	subject := fmt.Sprintf("Join %s in %s", data.OrgName, data.AppName)
	return commo.New(recipient, subject, "welcome_user", data)
}

// ============================================================================
// Reset password email
// ============================================================================

// ResetPasswordEmailData is used to complete the reset_password template.
type ResetPasswordEmailData struct {
	EmailBaseData
	ContactName         string                 // the user's name, if available
	PasswordLinkBaseURL *url.URL               // the app url
	Token               vero.VerificationToken // verification token for reset password link record
}

// VerifyURL returns the password-reset URL including the signed verification token.
func (d ResetPasswordEmailData) VerifyURL() string {
	if d.PasswordLinkBaseURL == nil {
		return ""
	}

	params := make(url.Values, 1)
	params.Set("token", d.Token.String())

	d.PasswordLinkBaseURL.RawQuery = params.Encode()
	return d.PasswordLinkBaseURL.String()
}

// NewResetPasswordEmail builds a reset_password commo email for the recipient.
func NewResetPasswordEmail(recipient string, data ResetPasswordEmailData) (*commo.Email, error) {
	subject := fmt.Sprintf("%s password reset request", data.AppName)
	return commo.New(recipient, subject, "reset_password", data)
}
