package emails_test

import (
	"bytes"
	"html/template"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/x/vero"
)

func TestVerifyWelcomeUserURL(t *testing.T) {
	invite := emails.WelcomeUserEmailData{
		PasswordResetURL: &url.URL{
			Scheme: "https",
			Host:   "resetpassword.example.com",
			Path:   "/reset-password",
		},
		Token: vero.VerificationToken("abc123"),
	}

	require.Equal(t, "https://resetpassword.example.com/reset-password?token=YWJjMTIz", invite.VerifyURL())
}

func TestWelcomeUserEmailBodyHTML(t *testing.T) {
	t.Run("RendersUnescaped", func(t *testing.T) {
		templates := emails.LoadTemplates()
		tmpl, ok := templates["welcome_user.html"]
		require.True(t, ok, "welcome_user.html template must exist")

		orgHomepage, _ := url.Parse("https://example.com")
		data := emails.WelcomeUserEmailData{
			EmailBaseData: emails.EmailBaseData{
				AppName:        "TestApp",
				OrgName:        "TestOrg",
				OrgHomepageURL: orgHomepage,
			},
			PasswordResetURL: &url.URL{
				Scheme: "https",
				Host:   "resetpassword.example.com",
				Path:   "/reset-password",
			},
			Token:                vero.VerificationToken("abc123"),
			WelcomeEmailBodyHTML: template.HTML("<p>Custom <strong>body</strong> content</p>"),
		}

		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, "base", data)
		require.NoError(t, err)

		html := buf.String()
		require.Contains(t, html, "<p>Custom <strong>body</strong> content</p>", "HTML in WelcomeEmailBodyHTML must be rendered as markup")
		require.NotContains(t, html, "&lt;p&gt;Custom &lt;strong&gt;body&lt;/strong&gt; content&lt;/p&gt;", "HTML must not be escaped")
	})

	t.Run("EmptyRendersDefaultMessage", func(t *testing.T) {
		templates := emails.LoadTemplates()
		tmpl, ok := templates["welcome_user.html"]
		require.True(t, ok, "welcome_user.html template must exist")

		orgHomepage, _ := url.Parse("https://example.com")
		data := emails.WelcomeUserEmailData{
			EmailBaseData: emails.EmailBaseData{
				AppName:        "TestApp",
				OrgName:        "TestOrg",
				OrgHomepageURL: orgHomepage,
			},
			PasswordResetURL: &url.URL{
				Scheme: "https",
				Host:   "resetpassword.example.com",
				Path:   "/reset-password",
			},
			Token: vero.VerificationToken("abc123"),
			// WelcomeEmailBodyHTML left zero so template uses {{ else }} branch
		}

		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, "base", data)
		require.NoError(t, err)

		html := buf.String()
		require.Contains(t, html, "You have been invited to join a team in TestApp", "default fallback message must appear when WelcomeEmailBodyHTML is empty")
	})
}

func TestVerifyResetPasswordURL(t *testing.T) {
	invite := emails.ResetPasswordEmailData{
		PasswordLinkBaseURL: &url.URL{
			Scheme: "https",
			Host:   "resetpassword.example.com",
			Path:   "/reset-password",
		},
		Token: vero.VerificationToken("abc123"),
	}

	require.Equal(t, "https://resetpassword.example.com/reset-password?token=YWJjMTIz", invite.VerifyURL())
}
