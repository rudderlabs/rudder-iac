package store

import (
	"crypto/sha256"
	"encoding/hex"
)

// clearKeyLen is the longest key a record keeps in clear: short keys are
// labels such as dev or api, not credentials.
const clearKeyLen = 8

// SetWriteKey stores key on rec. A key longer than 8 characters may be real:
// it is kept as its first 4 characters, its last 4 from 12 characters on,
// and its sha256.
func SetWriteKey(rec *Record, key string) {
	rec.WriteKey = key
	if len(key) <= clearKeyLen {
		return
	}
	rec.WriteKeySha256 = keySha256(key)
	rec.WriteKeyPrefix = key[:4]
	if len(key) >= 12 {
		rec.WriteKeySuffix = key[len(key)-4:]
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
