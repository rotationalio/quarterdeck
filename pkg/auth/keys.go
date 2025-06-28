package auth

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"

	"go.rtnl.ai/quarterdeck/pkg/errors"
)

const (
	BlockPublicKey  = "PUBLIC KEY"
	BlockPrivateKey = "PRIVATE KEY"
)

func GenerateKeys() (_ SigningKey, err error) {
	k := &keys{}
	if k.public, k.private, err = ed25519.GenerateKey(rand.Reader); err != nil {
		return nil, err
	}

	return k, nil
}

// SigningKey is an interface for cryptographic keys used for token signing without
// the need for callers to understand the specific signature algorithm.
type SigningKey interface {
	Load(path string) error
	Dump(path string) error
	PublicKey() crypto.PublicKey
	PrivateKey() crypto.PrivateKey
}

type keys struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

// Load the specified keys from the filesystem
// TODO: support loading keys from a vault or other secure storage.
func (k *keys) Load(path string) (err error) {
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return errors.Fmt("could not open %s: %w", path, err)
	}
	defer f.Close()

	for block, err := range pemBlocks(f) {
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		switch block.Type {
		case BlockPublicKey:
			if k.public != nil {
				return errors.Fmt("multiple public keys found in %s", path)
			}
			var pub any
			if pub, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
				return errors.Fmt("could not parse public key in %s: %w", path, err)
			}

			var ok bool
			if k.public, ok = pub.(ed25519.PublicKey); !ok {
				return errors.Fmt("public key in %s is not an ed25519 public key", path)
			}
		case BlockPrivateKey:
			if k.private != nil {
				return errors.Fmt("multiple private keys found in %s", path)
			}

			var prv any
			if prv, err = x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
				return errors.Fmt("could not parse private key in %s: %w", path, err)
			}

			var ok bool
			if k.private, ok = prv.(ed25519.PrivateKey); !ok {
				return errors.Fmt("private key in %s is not an ed25519 private key", path)
			}
		default:
			return errors.Fmt("unexpected PEM block type %q in %s", block.Type, path)
		}
	}

	if k.public == nil || k.private == nil {
		return errors.Fmt("missing public or private key in %s", path)
	}
	return nil
}

func (k *keys) Dump(path string) (err error) {
	var f *os.File
	if f, err = os.Create(path); err != nil {
		return errors.Fmt("could not create %s: %w", path, err)
	}
	defer f.Close()

	if k.private != nil {
		var der []byte
		if der, err = x509.MarshalPKCS8PrivateKey(k.private); err != nil {
			return errors.Fmt("could not marshal private key: %w", err)
		}

		if err = pem.Encode(f, &pem.Block{Type: BlockPrivateKey, Bytes: der}); err != nil {
			return errors.Fmt("could not write private key to %s: %w", path, err)
		}
	}

	if k.public != nil {
		var pkix []byte
		if pkix, err = x509.MarshalPKIXPublicKey(k.public); err != nil {
			return errors.Fmt("could not marshal public key: %w", err)
		}

		if err = pem.Encode(f, &pem.Block{Type: BlockPublicKey, Bytes: pkix}); err != nil {
			return errors.Fmt("could not write public key to %s: %w", path, err)
		}
	}

	return nil
}

func (k *keys) PublicKey() crypto.PublicKey {
	return k.public
}

func (k *keys) PrivateKey() crypto.PrivateKey {
	return k.private
}

func pemBlocks(f io.Reader) func(yield func(*pem.Block, error) bool) {
	return func(yield func(*pem.Block, error) bool) {
		data, err := io.ReadAll(f)
		if err != nil {
			yield(nil, errors.Fmt("failed to read from PEM reader: %w", err))
			return
		}

		for {
			var block *pem.Block
			block, data = pem.Decode(data)
			if block == nil {
				return
			}

			if !yield(block, nil) {
				return
			}
		}
	}
}
