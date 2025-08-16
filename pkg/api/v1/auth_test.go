package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/quarterdeck/pkg/api/v1"
)

func TestValidateLoginRequest(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		req := &LoginRequest{
			Email:    "gerard@example.com",
			Password: "password123!bangscat",
		}
		require.NoError(t, req.Validate(), "valid login request should not return an error")

	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			req *LoginRequest
			err string
		}{
			{
				req: &LoginRequest{
					Email:    "",
					Password: "password123!bangscat",
				},
				err: "missing email: this field is required",
			},
			{
				req: &LoginRequest{
					Email:    "gerard@example.com",
					Password: "",
				},
				err: "missing password: this field is required",
			},
			{
				req: &LoginRequest{
					Email:    "gerard@example.com",
					Password: "short",
				},
				err: "invalid field password: must be at least 8 characters long",
			},
		}

		for _, tc := range tests {
			err := tc.req.Validate()
			assert.Error(t, err, "expected error for invalid login request")
			assert.EqualError(t, err, tc.err, "expected specific error for invalid login request")
		}
	})
}

func TestValidateAuthenticateRequest(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		req := &AuthenticateRequest{
			ClientID:     "my-client-id",
			ClientSecret: "my-client-secret",
		}
		require.NoError(t, req.Validate(), "valid authenticate request should not return an error")
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			req *AuthenticateRequest
			err string
		}{
			{
				req: &AuthenticateRequest{
					ClientID:     "",
					ClientSecret: "my-client-secret",
				},
				err: "missing client_id: this field is required",
			},
			{
				req: &AuthenticateRequest{
					ClientID:     "my-client-id",
					ClientSecret: "",
				},
				err: "missing client_secret: this field is required",
			},
		}

		for _, tc := range tests {
			err := tc.req.Validate()
			assert.Error(t, err, "expected error for invalid authenticate request")
			assert.EqualError(t, err, tc.err, "expected specific error for invalid authenticate request")
		}
	})
}
