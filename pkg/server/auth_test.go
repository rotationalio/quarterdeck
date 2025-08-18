package server_test

import "testing"

func (s *serverTestSuite) TestAuthenticate() {
	s.T().Run("Success", func(t *testing.T) {
		// Request is successful and the response contains the expected tokens.
	})

	s.T().Run("BadRequest", func(t *testing.T) {
		// Request data is invalid
	})

	s.T().Run("KeyNotFound", func(t *testing.T) {
		// Client ID does not exist in the database
	})

	s.T().Run("VerificationFailed", func(t *testing.T) {
		// Client secret is wrong, should return unauthorized
	})
}

func (s *serverTestSuite) TestReauthenticate() {
	s.T().Run("ReauthUser", func(t *testing.T) {
		// Successful re-authentication of a user token
	})

	s.T().Run("ReauthAPIKey", func(t *testing.T) {
		// Successful re-authentication of an API key token
	})

	s.T().Run("BadRequest", func(t *testing.T) {
		// Request data is invalid
	})

	s.T().Run("TokenInvalid", func(t *testing.T) {
		// Refresh token is invalid or expired
	})

	s.T().Run("BadSubject", func(t *testing.T) {
		// Subject type is unknown
	})
}
