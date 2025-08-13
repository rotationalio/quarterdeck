package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"go.rtnl.ai/gimlet/cache"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/httpcc"
)

const (
	JWKSMaxAge               = 7 * 24 * time.Hour
	JWKSStaleWhileRevalidate = 24 * time.Hour
)

type JWKS struct {
	sync.RWMutex
	jose.JSONWebKeySet
	etag     string
	modified time.Time
	cc       string
	ccinit   sync.Once
}

// Append a key to the JWKS. If a key with the same KeyID already exists, an error is returned.
func (j *JWKS) Add(keyID ulid.ULID, key SigningKey) error {
	j.Lock()
	defer j.Unlock()

	kid := keyID.String()
	for _, existing := range j.Keys {
		if existing.KeyID == kid {
			return errors.Fmt("key with id %s already exists in the key set", kid)
		}
	}

	j.Keys = append(j.Keys, jose.JSONWebKey{
		Key:       key.PublicKey(),
		KeyID:     kid,
		Algorithm: signingMethod.Alg(),
		Use:       keyUse,
	})

	// If we've added a key, reset the ETag so it will be recomputed next time.
	j.etag = ""
	j.modified = time.Now().UTC()
	return nil
}

//===========================================================================
// ETagger Interface
//===========================================================================

var _ cache.ETagger = (*JWKS)(nil)

func (j *JWKS) ETag() string {
	j.RLock()
	if j.etag == "" {
		// Double checked locking
		j.RUnlock()
		j.computeETag()
		j.RLock()
	}
	defer j.RUnlock()
	return j.etag
}

func (j *JWKS) computeETag() (err error) {
	j.Lock()
	defer j.Unlock()

	// Double check locking from the ETag() method.
	if j.etag != "" {
		return nil
	}

	// If there are no keys, then return an empty ETag
	if len(j.Keys) == 0 {
		j.etag = ""
		return nil
	}

	// Compute the SHA256 hash of the JSON data and encode as hex
	hash := sha256.New()
	if err = json.NewEncoder(hash).Encode(j.JSONWebKeySet); err != nil {
		return fmt.Errorf("could not json encode jwks: %w", err)
	}
	j.etag = hex.EncodeToString(hash.Sum(nil))

	return nil
}

// ComputeETag to implement the ETagger interface but panics and should not be used.
func (j *JWKS) ComputeETag([]byte) {
	panic(errors.New("cannot compute etag on JWKS object only edit keys"))
}

// SetETag to implement the ETagger interface but panics and should not be used.
func (j *JWKS) SetETag(etag string) {
	panic(errors.New("cannot set etag on JWKS object only edit keys"))
}

//===========================================================================
// Expirer Interface
//===========================================================================

var _ cache.Expirer = (*JWKS)(nil)

func (j *JWKS) LastModified() time.Time {
	j.RLock()
	defer j.RUnlock()
	return j.modified
}

func (j *JWKS) Expires() time.Time {
	return time.Time{}
}

func (j *JWKS) Modified(time.Time, any) {
	panic(errors.New("cannot set modified timestamps on JWKS object only edit keys"))
}

//===========================================================================
// CacheController Interface
//===========================================================================

var _ cache.CacheController = (*JWKS)(nil)

func (j *JWKS) Directives() string {
	j.ccinit.Do(func() {
		builder := &httpcc.ResponseBuilder{
			StaleWhileRevalidate: uint64(JWKSStaleWhileRevalidate.Seconds()),
			MustRevalidate:       true,
			ProxyRevalidate:      true,
		}

		builder.SetMaxAge(uint64(JWKSMaxAge.Seconds()))
		builder.SetSMaxAge(uint64(JWKSMaxAge.Seconds()))
		j.cc = builder.String()
	})
	return j.cc
}

func (j *JWKS) SetMaxAge(any) {
	panic(errors.New("cannot set max age on JWKS object only edit keys"))
}

func (j *JWKS) SetSMaxAge(any) {
	panic(errors.New("cannot set s-max-age on JWKS object only edit keys"))
}
