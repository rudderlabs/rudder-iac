package store

import (
	"crypto/sha256"
	"encoding/hex"
)

// clearWriteKey is the listener's own key; a record keeps it in clear.
const clearWriteKey = "dev"

// SetWriteKey stores key on rec. A key that may be real is kept as its
// first and last 4 characters and its sha256; keys under 12 characters keep
// no characters at all.
func SetWriteKey(rec *Record, key string) {
	rec.WriteKey = key
	if key == "" || key == clearWriteKey {
		return
	}
	rec.WriteKeySha256 = keySha256(key)
	if len(key) >= 12 {
		rec.WriteKeyPrefix, rec.WriteKeySuffix = key[:4], key[len(key)-4:]
	}
	rec.WriteKey = rec.WriteKeyPrefix + "..." + rec.WriteKeySuffix
}

// MaskWriteKey is the form SetWriteKey stores for key.
func MaskWriteKey(key string) string {
	var rec Record
	SetWriteKey(&rec, key)
	return rec.WriteKey
}

// MatchesWriteKey reports whether rec was sent with the literal key.
func MatchesWriteKey(rec Record, key string) bool {
	if rec.WriteKeySha256 == "" {
		return rec.WriteKey == key
	}
	return rec.WriteKeySha256 == keySha256(key)
}

func keySha256(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
