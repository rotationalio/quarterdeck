package emails_test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/emails"
	"go.rtnl.ai/x/vero"
)

func TestVerifyResetPasswordURL(t *testing.T) {
	invite := emails.ResetPasswordEmailData{
		BaseURL: &url.URL{
			Scheme: "https",
			Host:   "resetpassword.example.com",
			Path:   "/reset-password",
		},
		Token: vero.VerificationToken("abc123"),
	}

	require.Equal(t, "https://resetpassword.example.com/reset-password?token=YWJjMTIz", invite.VerifyURL())
}
