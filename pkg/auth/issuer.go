package auth

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"net/url"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/gimlet/auth"
	"go.rtnl.ai/quarterdeck/pkg/auth/login"
	"go.rtnl.ai/quarterdeck/pkg/config"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/ulid"
)

// Global variables that should not be changed except between major versions.
var (
	signingMethod = jwt.SigningMethodEdDSA
	entropy       = ulid.Monotonic(rand.Reader, 1000)
	entropyMu     sync.Mutex
)

// Global constants that should not be changed except between major versions.
const (
	refreshPath = "/v1/reauthenticate"
	keyUse      = "sig"
)

type Issuer struct {
	conf            config.AuthConfig
	keyID           ulid.ULID
	key             crypto.PrivateKey
	publicKeys      *JWKS
	refreshAudience string
	loginURL        *login.URL
}

func NewIssuer(conf config.AuthConfig) (_ *Issuer, err error) {
	// Validate the issuer configuration
	if err = conf.Validate(); err != nil {
		return nil, err
	}

	issuer := &Issuer{
		conf:       conf,
		publicKeys: &JWKS{JSONWebKeySet: jose.JSONWebKeySet{Keys: make([]jose.JSONWebKey, 0, len(conf.Keys))}},
		loginURL:   login.New(conf.Issuer + "/login"),
	}

	// Load the specified keys from the filesystem.
	for kid, path := range conf.Keys {
		var keyID ulid.ULID
		if keyID, err = ulid.Parse(kid); err != nil {
			return nil, errors.Fmt("could not parse %s as a key id: %w", kid, err)
		}

		var keypair SigningKey
		if keypair, err = LoadKeys(path); err != nil {
			return nil, err
		}

		if err = issuer.AddKey(keyID, keypair); err != nil {
			return nil, errors.Fmt("could not add key %s: %w", kid, err)
		}
	}

	// If we have no keys, generate one for use (e.g. for testing or simple deployment)
	if issuer.key == nil {
		var keypair SigningKey
		if keypair, err = GenerateKeys(); err != nil {
			return nil, err
		}

		if err = issuer.AddKey(secureULID(), keypair); err != nil {
			return nil, errors.Fmt("could not add generated key: %w", err)
		}

		log.Warn().Str("keyID", issuer.keyID.String()).Msg("generated volatile claims issuer rsa key")
	}

	return issuer, nil
}

func SigningMethod() jwt.SigningMethod {
	return signingMethod
}

// Parse an access or refresh token verifying its signature but without verifying its
// claims. This ensures that valid JWT tokens are still accepted but claims can be
// handled on a case-by-case basis; for example by validating an expired access token
// during reauthentication.
func (tm *Issuer) Parse(tks string) (claims *auth.Claims, err error) {
	// TODO: will this still verify the signature?
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims = &auth.Claims{}
	if _, err = parser.ParseWithClaims(tks, claims, tm.GetKey); err != nil {
		return nil, err
	}
	return claims, nil
}

func (tm *Issuer) Sign(token *jwt.Token) (tks string, err error) {
	token.Header["kid"] = tm.keyID.String()
	return token.SignedString(tm.key)
}

func (tm *Issuer) CreateAccessToken(claims *auth.Claims) (_ *jwt.Token, err error) {
	now := time.Now()
	sub := claims.RegisteredClaims.Subject

	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        secureULID().String(),
		Subject:   sub,
		Audience:  jwt.ClaimStrings(tm.conf.Audience),
		Issuer:    tm.conf.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tm.conf.AccessTokenTTL)),
	}

	return jwt.NewWithClaims(signingMethod, claims), nil
}

func (tm *Issuer) CreateRefreshToken(accessToken *jwt.Token) (_ *jwt.Token, err error) {
	accessClaims, ok := accessToken.Claims.(*auth.Claims)
	if !ok {
		return nil, errors.ErrUnparsableClaims
	}

	// Add the refresh audience to the audience claims
	audience := append(accessClaims.Audience, tm.RefreshAudience())

	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessClaims.ID,
			Audience:  audience,
			Issuer:    accessClaims.Issuer,
			Subject:   accessClaims.Subject,
			IssuedAt:  accessClaims.IssuedAt,
			NotBefore: jwt.NewNumericDate(accessClaims.ExpiresAt.Add(tm.conf.TokenOverlap)),
			ExpiresAt: jwt.NewNumericDate(accessClaims.IssuedAt.Add(tm.conf.RefreshTokenTTL)),
		},
	}

	return jwt.NewWithClaims(signingMethod, claims), nil
}

// CreateTokens creates and signs an access and refresh token in one step.
func (tm *Issuer) CreateTokens(claims *auth.Claims) (signedAccessToken, signedRefreshToken string, err error) {
	var accessToken, refreshToken *jwt.Token

	if accessToken, err = tm.CreateAccessToken(claims); err != nil {
		return "", "", fmt.Errorf("could not create access token: %w", err)
	}

	if refreshToken, err = tm.CreateRefreshToken(accessToken); err != nil {
		return "", "", fmt.Errorf("could not create refresh token: %w", err)
	}

	if signedAccessToken, err = tm.Sign(accessToken); err != nil {
		return "", "", fmt.Errorf("could not sign access token: %w", err)
	}

	if signedRefreshToken, err = tm.Sign(refreshToken); err != nil {
		return "", "", fmt.Errorf("could not sign refresh token: %w", err)
	}

	return signedAccessToken, signedRefreshToken, nil
}

// Keys returns the map of ulid to public key for use externally.
func (tm *Issuer) Keys() (_ *JWKS, err error) {
	if len(tm.publicKeys.Keys) == 0 {
		return nil, errors.ErrNoSigningKeys
	}
	return tm.publicKeys, nil
}

// CurrentKey returns the ulid of the current key being used to sign tokens.
func (tm *Issuer) CurrentKey() ulid.ULID {
	return tm.keyID
}

// AddKey adds a new key to the issuer and updates the current key if the new is newer
// than the current key. The keyID must be a valid ULID and the ULID timestamp must
// fall after the current key's timestamp.
func (tm *Issuer) AddKey(keyID ulid.ULID, key SigningKey) (err error) {
	if err = tm.publicKeys.Add(keyID, key); err != nil {
		return err
	}

	if tm.key == nil || keyID.Time() > tm.keyID.Time() {
		tm.key = key.PrivateKey()
		tm.keyID = keyID
	}

	return nil
}

// Computes the refresh audience claim based on the issuer URL and a specific path to
// better protect refresh tokens from being used in other contexts.
func (tm *Issuer) RefreshAudience() string {
	if tm.refreshAudience == "" {
		if aud, err := url.Parse(tm.conf.Issuer); err == nil && tm.conf.Issuer != "" {
			tm.refreshAudience = aud.ResolveReference(&url.URL{Path: refreshPath}).String()
		} else {
			// The issuer URL should have been validated in the config.
			panic("could not parse issuer URL: " + err.Error())
		}
	}
	return tm.refreshAudience
}

// GetKey is an jwt.KeyFunc that selects the public key from the list of managed
// internal keys based on the kid in the token header. If the kid does not exist an
// error is returned and the token will not be able to be verified.
func (tm *Issuer) GetKey(token *jwt.Token) (key interface{}, err error) {
	// Per JWT security notice: do not forget to validate alg is expected
	if token.Method.Alg() != signingMethod.Alg() {
		return nil, errors.Fmt("unexpected signing method: %v", token.Method.Alg())
	}

	// Fetch the kid from the header
	kid, ok := token.Header["kid"]
	if !ok {
		return nil, errors.ErrNoKeyID
	}

	// Parse the kid
	var keyID ulid.ULID
	if keyID, err = ulid.Parse(kid.(string)); err != nil {
		return nil, errors.Fmt("could not parse kid: %w", err)
	}

	if keyID.IsZero() {
		return nil, errors.ErrInvalidKeyID
	}

	// Fetch the key from the list of managed keys
	keys := tm.publicKeys.Key(keyID.String())
	if len(keys) == 0 {
		return nil, errors.ErrUnknownSigningKey
	}

	// If we have multiple keys, return the first one; this should not happen
	if len(keys) > 1 {
		log.Warn().Str("keyID", keyID.String()).
			Msg("multiple signing keys found for kid")
	}

	return keys[0].Key, nil
}

func secureULID() ulid.ULID {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Now(), entropy)
}
