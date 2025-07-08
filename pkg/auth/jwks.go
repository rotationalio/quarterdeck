package auth

import (
	"crypto/sha256"
	"encoding/hex"

	jose "github.com/go-jose/go-jose/v4"
	"go.rtnl.ai/quarterdeck/pkg/errors"
	"go.rtnl.ai/ulid"
)

type JWKS struct {
	jose.JSONWebKeySet
	etag string
}

func (j *JWKS) ETag() (_ string, err error) {
	if j.etag == "" {
		hash := sha256.New()
		for _, key := range j.Keys {
			var keyID ulid.ULID
			if keyID, err = ulid.Parse(key.KeyID); err != nil {
				return "", errors.Fmt("could not parse key id %q: %w", key.KeyID, err)
			}

			if _, err = hash.Write(keyID[:]); err != nil {
				return "", errors.Fmt("could not write key id %q to hash: %w", key.KeyID, err)
			}

			j.etag = hex.EncodeToString(hash.Sum(nil))
		}
	}
	return j.etag, nil
}

// Append a key to the JWKS. If a key with the same KeyID already exists, an error is returned.
func (j *JWKS) Add(keyID ulid.ULID, key SigningKey) error {
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
	return nil
}
