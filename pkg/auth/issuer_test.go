package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
	"go.rtnl.ai/gimlet/auth"
	. "go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/quarterdeck/pkg/errors"

	"go.rtnl.ai/quarterdeck/pkg/config"
)

type TokenTestSuite struct {
	suite.Suite
	testdata map[string]string
}

func (s *TokenTestSuite) SetupSuite() {
	// Create the keys map from the testdata directory to create new token managers.
	s.testdata = make(map[string]string)
	s.testdata["01JYSHGWTSMK34J100N2Q0D21C"] = "testdata/01JYSHGWTSMK34J100N2Q0D21C.pem"
	s.testdata["01JYSW0C9QK2TN3MQ1T7F411DX"] = "testdata/01JYSW0C9QK2TN3MQ1T7F411DX.pem"
}

func (s *TokenTestSuite) AuthConfig() config.AuthConfig {
	// Helper function to return a default auth config for the tests.
	return config.AuthConfig{
		Keys:            s.testdata,
		Audience:        []string{"http://localhost:3000"},
		Issuer:          "http://localhost:3001",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}
}

func (s *TokenTestSuite) TestClaimsIssuer() {
	require := s.Require()
	conf := s.AuthConfig()

	tm, err := NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	s.Run("KeyLoading", func() {
		// Check that the keys are loaded correctly and the latest key is set as the current key.
		keys, err := tm.Keys()
		require.NoError(err, "could not fetch jwks from issuer")
		require.Len(keys.Keys, 2)
		require.Equal("01JYSW0C9QK2TN3MQ1T7F411DX", tm.CurrentKey().String())
	})

	s.Run("TokenIssuance", func() {
		// Create an access token from simple claims
		creds := &auth.Claims{
			Email: "kate@example.com",
			Name:  "Kate Holland",
		}

		accessToken, err := tm.CreateAccessToken(creds)
		require.NoError(err, "could not create access token from claims")
		require.IsType(&auth.Claims{}, accessToken.Claims)

		time.Sleep(500 * time.Millisecond)
		now := time.Now()

		// Check access token claims
		ac := accessToken.Claims.(*auth.Claims)
		require.NotZero(ac.ID)
		require.Equal(jwt.ClaimStrings{"http://localhost:3000"}, ac.Audience)
		require.Equal("http://localhost:3001", ac.Issuer)
		require.True(ac.IssuedAt.Before(now))
		require.True(ac.NotBefore.Before(now))
		require.True(ac.ExpiresAt.After(now))
		require.Equal(creds.Email, ac.Email)
		require.Equal(creds.Name, ac.Name)

		// Create a refresh token from the access token
		refreshToken, err := tm.CreateRefreshToken(accessToken)
		require.NoError(err, "could not create refresh token from access token")
		require.IsType(&auth.Claims{}, refreshToken.Claims)

		// Check refresh token claims
		// Check access token claims
		rc := refreshToken.Claims.(*auth.Claims)
		require.Equal(ac.ID, rc.ID, "access and refresh tokens must have same jid")
		require.NotEqual(ac.Audience, rc.Audience, "expected refresh token to have refresh audience")
		require.Equal(jwt.ClaimStrings{"http://localhost:3000", "http://localhost:3001/v1/reauthenticate"}, rc.Audience)
		require.Equal(ac.Issuer, rc.Issuer)
		require.Equal(ac.Subject, rc.Subject)
		require.True(rc.IssuedAt.Equal(ac.IssuedAt.Time))
		require.True(rc.NotBefore.After(now))
		require.True(rc.ExpiresAt.After(rc.NotBefore.Time))
		require.Empty(rc.Email)

		// Verify relative nbf and exp claims of access and refresh tokens
		require.True(ac.IssuedAt.Equal(rc.IssuedAt.Time), "access and refresh tokens do not have same iss timestamp")
		require.Equal(45*time.Minute, rc.NotBefore.Sub(ac.IssuedAt.Time), "refresh token nbf is not 45 minutes after access token iss")
		require.Equal(15*time.Minute, ac.ExpiresAt.Sub(rc.NotBefore.Time), "refresh token active does not overlap active token active by 15 minutes")
		require.Equal(60*time.Minute, rc.ExpiresAt.Sub(ac.ExpiresAt.Time), "refresh token does not expire 1 hour after access token")

		// Sign the access token
		atks, err := tm.Sign(accessToken)
		require.NoError(err, "could not sign access token")

		// Sign the refresh token
		rtks, err := tm.Sign(refreshToken)
		require.NoError(err, "could not sign refresh token")
		require.NotEqual(atks, rtks, "identical access and refresh tokens")

		// Validate the access token
		_, err = tm.Verify(atks)
		require.NoError(err, "could not validate access token")

		// Validate the refresh token (should be invalid because of not before in the future)
		_, err = tm.Verify(rtks)
		require.Error(err, "refresh token is valid?")
	})
}

func (s *TokenTestSuite) TestKeysGenerated() {
	require := s.Require()
	conf := s.AuthConfig()
	conf.Keys = nil

	// Create the token manager
	tm, err := NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Check that the keys are generated
	keys, err := tm.Keys()
	require.NoError(err, "could not fetch jwks from issuer")
	require.Len(keys.Keys, 1)
}

func (s *TokenTestSuite) TestValidTokens() {
	require := s.Require()
	conf := s.AuthConfig()
	conf.TokenOverlap = -1 * conf.AccessTokenTTL

	tm, err := NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Default creds
	creds := &auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	}

	accessToken, refreshToken, err := tm.CreateTokens(creds)
	require.NoError(err)

	_, err = tm.Verify(accessToken)
	require.NoError(err, "could not verify access token")

	// Should be valid because access token overlap is equal to access token TTL.
	_, err = tm.Verify(refreshToken)
	require.NoError(err, "could not verify refresh token")
}

func (s *TokenTestSuite) TestInvalidTokens() {
	// Create the token manager
	require := s.Require()
	conf := s.AuthConfig()

	tm, err := NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Manually create a token to validate with the token manager
	now := time.Now()

	// Helper function to create a token with specified claims and modify for tests.
	makeToken := func(fields map[string]any) string {
		// This would be valid claims if not modified by fields map.
		claims := &auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ID:        "01JYKWYN0AFYNEPG01VQZYZ8MA",
				Subject:   "u01JYKX4BFXQCSDDFSW7D47491B",
				Issuer:    conf.Issuer,
				Audience:  jwt.ClaimStrings(conf.Audience),
				IssuedAt:  jwt.NewNumericDate(now.Add(-1 * time.Hour)),
				NotBefore: jwt.NewNumericDate(now.Add(-1 * time.Hour)),
				ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			},
			Email: "kate@example.com",
			Name:  "Kate Holland",
		}

		for k, v := range fields {
			switch k {
			case "id":
				claims.ID = v.(string)
			case "sub":
				claims.Subject = v.(string)
			case "iss":
				claims.Issuer = v.(string)
			case "aud":
				claims.Audience = jwt.ClaimStrings{v.(string)}
			case "iat":
				claims.IssuedAt = jwt.NewNumericDate(v.(time.Time))
			case "nbf":
				claims.NotBefore = jwt.NewNumericDate(v.(time.Time))
			case "exp":
				claims.ExpiresAt = jwt.NewNumericDate(v.(time.Time))
			case "clientID":
				claims.ClientID = v.(string)
			case "name":
				claims.Name = v.(string)
			case "email":
				claims.Email = v.(string)
			case "gravatar":
				claims.Gravatar = v.(string)
			case "role":
				claims.Roles = v.([]string)
			case "permissions":
				claims.Permissions = v.([]string)
			}
		}

		var (
			ok            bool
			signingMethod jwt.SigningMethod
		)
		if signingMethod, ok = fields["signingMethod"].(jwt.SigningMethod); !ok {
			signingMethod = jwt.SigningMethodEdDSA
		}

		token := jwt.NewWithClaims(signingMethod, claims)
		if kid, ok := fields["kid"].(string); ok {
			token.Header["kid"] = kid
		}

		if genKey, ok := fields["genKey"].(bool); ok && genKey {
			if signingMethod.Alg() == jwt.SigningMethodEdDSA.Alg() {
				key, err := GenerateKeys()
				require.NoError(err, "could not generate signing keys")

				tks, err := token.SignedString(key.PrivateKey())
				require.NoError(err, "could not sign token with generated keys")
				return tks
			}

			if signingMethod.Alg() == jwt.SigningMethodRS256.Alg() {
				key, err := rsa.GenerateKey(rand.Reader, 4096)
				require.NoError(err, "could not generate RSA signing keys")

				tks, err := token.SignedString(key)
				require.NoError(err, "could not sign token with generated keys")
				return tks
			}

			require.Fail("unknown signing method for generated keys", signingMethod.Alg())
		}

		tks, err := tm.Sign(token)
		require.NoError(err, "could not sign token with claims")
		return tks
	}

	testCases := []struct {
		tks string
		err error
		msg string
	}{
		{
			tks: "crazystring",
			err: jwt.ErrTokenMalformed,
			msg: "not a jwt token at all",
		},
		{
			tks: "",
			err: jwt.ErrTokenMalformed,
			msg: "empty string",
		},
		{
			tks: makeToken(nil),
			err: nil,
			msg: "token should be valid with no modification",
		},
		{
			tks: makeToken(map[string]any{"kid": "01GE63H600NKHE7B8Y7MHW1VGV", "genKey": true}),
			err: errors.ErrUnknownSigningKey,
			msg: "signed with unknown kid and generatred keys",
		},
		{
			tks: makeToken(map[string]any{"kid": "01GE62EXXR0X0561XD53RDFBQJ", "genKey": true}),
			err: jwt.ErrTokenUnverifiable,
			msg: "signed with known kid but wrong keys",
		},
		{
			tks: makeToken(map[string]any{"signingMethod": jwt.SigningMethodRS256, "genKey": true}),
			err: jwt.ErrTokenSignatureInvalid,
			msg: "incorrect signing method used",
		},
		{
			tks: makeToken(map[string]any{"nbf": now.Add(1 * time.Hour)}),
			err: jwt.ErrTokenNotValidYet,
			msg: "time based verification: not before in the future",
		},
		{
			tks: makeToken(map[string]any{"iat": now.Add(1 * time.Hour)}),
			err: nil,
			msg: "issued at should not be verified",
		},
		{
			tks: makeToken(map[string]any{"exp": now.Add(-1 * time.Hour)}),
			err: jwt.ErrTokenExpired,
			msg: "expires at is in the past",
		},
		{
			tks: makeToken(map[string]any{"aud": "http://foo.com"}),
			err: jwt.ErrTokenInvalidAudience,
			msg: "invalid audience",
		},
		{
			tks: makeToken(map[string]any{"iss": "http://foo.com"}),
			err: jwt.ErrTokenInvalidIssuer,
			msg: "invalid issuer",
		},
	}

	for _, tc := range testCases {
		_, err := tm.Verify(tc.tks)
		require.ErrorIs(err, tc.err, "unexpected error for case: %s", tc.msg)
	}
}

// Test that a token signed with rotated keys can still be verified.
// This also tests that the correct signing key is required.
func (s *TokenTestSuite) TestKeyRotation() {
	require := s.Require()

	// Create the "old claims issuer"
	testdata := make(map[string]string)
	testdata["01JYSHGWTSMK34J100N2Q0D21C"] = "testdata/01JYSHGWTSMK34J100N2Q0D21C.pem"

	conf := s.AuthConfig()
	conf.Keys = testdata

	oldTM, err := NewIssuer(conf)
	require.NoError(err, "could not initialize old token manager")

	// Create the "new" claims issuer with the new key
	conf2 := s.AuthConfig()
	newTM, err := NewIssuer(conf2)
	require.NoError(err, "could not initialize new token manager")

	// Create a valid token with the "old claims issuer"
	token, err := oldTM.CreateAccessToken(&auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	})
	require.NoError(err)

	tks, err := oldTM.Sign(token)
	require.NoError(err)

	// Validate token with "new claims issuer"
	_, err = newTM.Verify(tks)
	require.NoError(err)

	// A token created by the "new claims issuer" should not be verified by the old one.
	tks, err = newTM.Sign(token)
	require.NoError(err)

	_, err = oldTM.Verify(tks)
	require.Error(err)
}

// Test that a token can be parsed even if it is expired. This is necessary to parse
// access tokens in order to use a refresh token to extract the claims.
func (s *TokenTestSuite) TestParseExpiredToken() {
	require := s.Require()
	conf := config.AuthConfig{
		Keys:            s.testdata,
		Audience:        []string{"http://localhost:3000"},
		Issuer:          "http://localhost:3001",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	tm, err := NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Default creds
	creds := &auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	}

	accessToken, err := tm.CreateAccessToken(creds)
	require.NoError(err, "could not create access token from claims")
	require.IsType(&auth.Claims{}, accessToken.Claims)

	// Modify claims to be expired
	claims := accessToken.Claims.(*auth.Claims)
	claims.IssuedAt = jwt.NewNumericDate(claims.IssuedAt.Add(-24 * time.Hour))
	claims.ExpiresAt = jwt.NewNumericDate(claims.ExpiresAt.Add(-24 * time.Hour))
	claims.NotBefore = jwt.NewNumericDate(claims.NotBefore.Add(-24 * time.Hour))
	accessToken.Claims = claims

	// Create signed token
	tks, err := tm.Sign(accessToken)
	require.NoError(err, "could not create expired access token from claims")

	// Ensure that verification fails; claims are invalid.
	pclaims, err := tm.Verify(tks)
	require.Error(err, "expired token was somehow validated?")
	require.Empty(pclaims, "verify returned claims even after error")

	// Parse token without verifying claims but verifying the signature
	pclaims, err = tm.Parse(tks)
	require.NoError(err, "claims were validated in parse")
	require.NotEmpty(pclaims, "parsing returned empty claims without error")

	// Check claims
	require.Equal(claims.ID, pclaims.ID)
	require.Equal(claims.ExpiresAt, pclaims.ExpiresAt)
	require.Equal(creds.Email, claims.Email)
	require.Equal(creds.Name, claims.Name)

	// Ensure signature is still validated on parse
	tks += "abcdefg"
	claims, err = tm.Parse(tks)
	require.Error(err, "claims were parsed with bad signature")
	require.Empty(claims, "bad signature token returned non-empty claims")
}

func (s *TokenTestSuite) TestRefreshAudience() {
	require := s.Require()

	s.Run("FromIssuer", func() {
		conf := s.AuthConfig()
		conf.Issuer = "https://auth.rotational.app"

		tm, err := NewIssuer(conf)
		require.NoError(err, "could not initialize token manager")

		audience := tm.RefreshAudience()
		require.Equal("https://auth.rotational.app/v1/reauthenticate", audience, "refresh audience does not match expected value")
	})

	s.Run("Panics", func() {
		// Cannot use NewIssuer because it requires a valid config
		tm := &Issuer{}
		require.Panics(func() {
			_ = tm.RefreshAudience()
		})
	})
}

func (s *TokenTestSuite) TestGetKeyErrors() {
	require := s.Require()
	conf := s.AuthConfig()
	tm, err := NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	tests := []struct {
		token *jwt.Token
		err   string
	}{
		{
			token: &jwt.Token{
				Header: map[string]any{"kid": "01JYSHGWTSMK34J100N2Q0D21C"},
				Method: jwt.SigningMethodNone,
			},
			err: "unexpected signing method: none",
		},
		{
			token: &jwt.Token{
				Header: map[string]any{"kid": "\x000000foo"},
				Method: jwt.SigningMethodEdDSA,
			},
			err: "could not parse kid: ulid: bad data size when unmarshaling",
		},
		{
			token: &jwt.Token{
				Method: jwt.SigningMethodEdDSA,
			},
			err: errors.ErrNoKeyID.Error(),
		},
		{
			token: &jwt.Token{
				Header: map[string]any{"kid": "00000000000000000000000000"},
				Method: jwt.SigningMethodEdDSA,
			},
			err: errors.ErrInvalidKeyID.Error(),
		},
		{
			token: &jwt.Token{
				Header: map[string]any{"kid": "01JZNMMNJ4SR4DM9ZBMSSXHJEK"},
				Method: jwt.SigningMethodEdDSA,
			},
			err: errors.ErrUnknownSigningKey.Error(),
		},
	}

	for i, tc := range tests {
		key, err := tm.GetKey(tc.token)
		require.EqualError(err, tc.err, "expected error for test case %d", i)
		require.Nil(key, "expected nil key for test case %d", i)
	}

}

func (s *TokenTestSuite) TestAlgorithm() {
	// Ensure the JWKS key algorithm constant is set correctly between libraries.
	// We use go-jose for JWKS and golang-jwt for JWT tokens, so the algorithm must match.
	require := s.Require()
	require.Equal(SigningMethod().Alg(), string(jose.EdDSA), "go-jose and golang-jwt signing methods do not match")
}

// Execute suite as a go test.
func TestTokenTestSuite(t *testing.T) {
	suite.Run(t, new(TokenTestSuite))
}
