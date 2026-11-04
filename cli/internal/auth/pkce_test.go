package auth

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var base64URL = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func TestNewCodeVerifier(t *testing.T) {
	t.Parallel()

	first, err := newCodeVerifier()
	require.NoError(t, err)
	second, err := newCodeVerifier()
	require.NoError(t, err)

	assert.Len(t, first, 43)
	assert.Regexp(t, base64URL, first)
	assert.NotEqual(t, first, second)
}

// The vector is RFC 7636 appendix B, so config-backend computes the same value.
func TestCodeChallengeS256_RFC7636Vector(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
		codeChallengeS256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"),
	)
}

func TestCodeChallengeS256_HasNoPadding(t *testing.T) {
	t.Parallel()

	verifier, err := newCodeVerifier()
	require.NoError(t, err)

	challenge := codeChallengeS256(verifier)
	assert.Len(t, challenge, 43)
	assert.Regexp(t, base64URL, challenge)
}

func TestNewState(t *testing.T) {
	t.Parallel()

	first, err := newState()
	require.NoError(t, err)
	second, err := newState()
	require.NoError(t, err)

	assert.Len(t, first, 22)
	assert.Regexp(t, base64URL, first)
	assert.NotEqual(t, first, second)
}
