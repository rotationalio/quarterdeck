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
			WelcomeEmailBodyHTML: template.HTML("<p>{{ .OrgName }} <strong>{{ .AppName }}</strong> {{ .OrgHomepageURL }}</p>"),
		}

		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, "base", data)
		require.NoError(t, err)

		html := buf.String()
		require.NotContains(t, html, "{{ .OrgName }}", "OrgName must not be rendered as text")
		require.NotContains(t, html, "{{ .AppName }}", "AppName must not be rendered as text")
		require.NotContains(t, html, "{{ .OrgHomepageURL }}", "OrgHomepageURL must not be rendered as text")
		require.NotContains(t, html, "&lt;p&gt;TestOrg &lt;strong&gt;TestApp&lt;/strong&gt; https://example.com&lt;/p&gt;", "HTML must not be escaped")
		require.Contains(t, html, "<p>TestOrg <strong>TestApp</strong> https://example.com</p>", "HTML in WelcomeEmailBodyHTML must be rendered as markup")
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
