package emails_test

import (
	"html/template"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/commo"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/x/vero"
)

// TestValidateWelcomeUserEmailHappy ensures welcome templates render with Role in the body.
func TestValidateWelcomeUserEmailHappy(t *testing.T) {
	commo.WithTemplates(emails.LoadTemplates())

	orgHomepage, _ := url.Parse("https://example.com")
	resetURL, _ := url.Parse("https://app.example.com/reset-password")

	data := emails.WelcomeUserEmailData{
		EmailBaseData: emails.EmailBaseData{
			AppName:        "TestApp",
			OrgName:        "TestOrg",
			OrgHomepageURL: orgHomepage,
		},
		Role:             "Analyst",
		PasswordResetURL: resetURL,
		Token:            vero.VerificationToken("abc123"),
		WelcomeEmailBodyText: "Role: {{ if .Role }}{{ .Role }}{{ else }}Team Member{{ end }}",
		WelcomeEmailBodyHTML: template.HTML(
			`<p>Role: {{ if .Role }}{{ .Role }}{{ else }}Team Member{{ end }}</p>`,
		),
	}

	require.NoError(t, emails.ValidateWelcomeUserEmail(data))
}

// TestValidateWelcomeUserEmailRendersEmptyBody ensures an error is returned if
// the welcome email body text or html is empty.
func TestValidateWelcomeUserEmailRendersEmptyBody(t *testing.T) {
	commo.WithTemplates(emails.LoadTemplates())

	data := emails.WelcomeUserEmailData{
		EmailBaseData: emails.EmailBaseData{
			AppName: "TestApp",
		},
	}

	require.ErrorIs(t, emails.ValidateWelcomeUserEmail(data), errors.ErrEmptyWelcomeEmailBody)
}
