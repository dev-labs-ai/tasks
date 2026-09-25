package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// TokenBytes is the size of a session token before encoding.
const TokenBytes = 32

// NewSessionToken returns a random token for the session cookie and the
// hash to store in the database. The database never sees the token itself,
// so a leaked sessions table cannot be used to log in.
func NewSessionToken() (token string, hash []byte) {
	b := make([]byte, TokenBytes)
	_, _ = rand.Read(b) // never fails; see crypto/rand.Read
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashSessionToken(token)
}

// HashSessionToken returns the SHA-256 of a cookie token, as stored in the
// sessions table.
func HashSessionToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
