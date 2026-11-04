package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// randomString returns n random bytes as unpadded base64url.
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// newCodeVerifier returns a PKCE verifier. 32 bytes encode to 43 characters,
// the RFC 7636 minimum, which the webapp and config-backend both check.
func newCodeVerifier() (string, error) {
	return randomString(32)
}

func codeChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func newState() (string, error) {
	return randomString(16)
}
