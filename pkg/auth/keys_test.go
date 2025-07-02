package auth_test

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/quarterdeck/pkg/auth"
)

func TestSigningKeys(t *testing.T) {
	// Generating signing keys should not error
	keypair, err := GenerateKeys()
	require.NoError(t, err)

	t.Run("PublicKey", func(t *testing.T) {
		// Set to the current configured algorithm for token signing
		// NOTE: change expected type if the algorithm changes
		pubKey := keypair.PublicKey()
		require.NotNil(t, pubKey, "PublicKey should not be nil")
		require.IsType(t, ed25519.PublicKey{}, pubKey, "PublicKey should be of type ed25519.PublicKey")
	})

	t.Run("PrivateKey", func(t *testing.T) {
		// Set to the current configured algorithm for token signing
		// NOTE: change expected type if the algorithm changes
		privKey := keypair.PrivateKey()
		require.NotNil(t, privKey, "PrivateKey should not be nil")
		require.IsType(t, ed25519.PrivateKey{}, privKey, "PrivateKey should be of type ed25519.PrivateKey")
	})

	t.Run("Serialize", func(t *testing.T) {
		// Test saving the key to disk then loading it again
		path := t.TempDir() + "/testkey.pem"
		require.NoError(t, keypair.Dump(path), "could not save key to disk")
		require.FileExists(t, path, "no file was created at the expected path")

		// Should be able to load the key back from disk
		cmpt, err := LoadKeys(path)
		require.NoError(t, err, "could not load key from disk")
		require.Equal(t, keypair.PublicKey(), cmpt.PublicKey(), "loaded public key does not match original")
		require.Equal(t, keypair.PrivateKey(), cmpt.PrivateKey(), "loaded private key does not match original")
	})
}
