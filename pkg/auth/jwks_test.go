package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/quarterdeck/pkg/auth"
	"go.rtnl.ai/ulid"
)

func TestJWKS(t *testing.T) {
	// Load keys from disk
	keys := make([]auth.SigningKey, 0, 2)
	for _, path := range []string{"testdata/01JYSHGWTSMK34J100N2Q0D21C.pem", "testdata/01JYSW0C9QK2TN3MQ1T7F411DX.pem"} {
		key, err := auth.LoadKeys(path)
		require.NoError(t, err, "could not load key from %s", path)
		keys = append(keys, key)
	}

	t.Run("Add", func(t *testing.T) {
		jwks := &auth.JWKS{}
		for _, key := range keys {
			require.NoError(t, jwks.Add(ulid.Make(), key), "could not add key")
		}

		require.Len(t, jwks.Keys, len(keys), "expected %d keys in JWKS", len(keys))
	})

	t.Run("AddDuplicate", func(t *testing.T) {
		jwks := &auth.JWKS{}
		keyID := ulid.Make()
		err := jwks.Add(keyID, keys[0])
		require.NoError(t, err, "could not add key")

		err = jwks.Add(keyID, keys[0])
		require.Error(t, err, "expected error when adding duplicate key")
		require.EqualError(t, err, "key with id "+keyID.String()+" already exists in the key set", "unexpected error message")
	})

	t.Run("ETag", func(t *testing.T) {
		jwks := &auth.JWKS{}
		for _, key := range keys {
			require.NoError(t, jwks.Add(ulid.Make(), key), "could not add key")
		}

		etag, err := jwks.ETag()
		require.NoError(t, err, "could not compute ETag")
		require.NotEmpty(t, etag, "expected non-empty ETag")

		// Test caching/duplicate etag when no changes are made
		etag2, err := jwks.ETag()
		require.NoError(t, err, "could not compute ETag again")
		require.Equal(t, etag, etag2, "ETag should not change if no keys are added or removed")

		// Add another key and check if ETag changes
		// NOTE: requires on the assumption that duplicate keys are only checked by keyID.
		require.NoError(t, jwks.Add(ulid.Make(), keys[0]), "could not add key")

		etag3, err := jwks.ETag()
		require.NoError(t, err, "could not compute ETag again")
		require.NotEqual(t, etag, etag3, "ETag should change if a new key is added")
	})

	t.Run("ETagEmpty", func(t *testing.T) {
		jwks := &auth.JWKS{}
		etag, err := jwks.ETag()
		require.NoError(t, err, "could not compute ETag for empty JWKS")
		require.Empty(t, etag, "expected empty ETag for empty JWKS")
	})
}
