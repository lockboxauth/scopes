package auth

import (
	"crypto/rsa"

	"github.com/pkg/errors"
	jose "gopkg.in/square/go-jose.v2"
)

// Parse verifies the signed payload is accurate for the supplied
// public key, and returns the contents of the payload.
func Parse(signedPayload, publicKey string) (string, error) {
	object, err := jose.ParseSigned(signedPayload)
	if err != nil {
		return "", errors.Wrap(err, "error parsing payload")
	}
	out, err := object.Verify(&publicKey)
	if err != nil {
		return "", errors.Wrap(err, "error verifying payload")
	}
	return string(out), nil
}

// Sign signs the passed payload with the passed private key, returning
// a compact serialization suitable for use with Parse.
func Sign(payload string, key *rsa.PrivateKey) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: key}, nil)
	if err != nil {
		return "", errors.Wrap(err, "error instantiating signer")
	}
	object, err := signer.Sign([]byte(payload))
	if err != nil {
		return "", errors.Wrap(err, "error signing payload")
	}
	out, err := object.CompactSerialize()
	if err != nil {
		return "", errors.Wrap(err, "error serializing signed payload")
	}
	return string(out), nil
}
