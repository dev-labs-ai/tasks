package auth

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	encoded := HashPassword("correct horse battery staple")

	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("hash %q does not carry the default parameters", encoded)
	}

	ok, err := VerifyPassword("correct horse battery staple", encoded)
	if err != nil || !ok {
		t.Errorf("right password: ok = %v, err = %v; want true", ok, err)
	}

	ok, err = VerifyPassword("correct horse battery stapler", encoded)
	if err != nil || ok {
		t.Errorf("wrong password: ok = %v, err = %v; want false", ok, err)
	}

	if other := HashPassword("correct horse battery staple"); other == encoded {
		t.Error("two hashes of one password are equal; the salt is not random")
	}
}

func TestVerifyPasswordReadsParametersFromTheHash(t *testing.T) {
	cheap := Params{Memory: 8192, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}
	encoded := hash("secret password", cheap)

	ok, err := VerifyPassword("secret password", encoded)
	if err != nil || !ok {
		t.Fatalf("hash with non-default parameters: ok = %v, err = %v; want true", ok, err)
	}

	// Same salt and key, different stated cost: the derived key changes, so
	// the password no longer matches.
	tampered := strings.Replace(encoded, "t=1", "t=2", 1)
	ok, err = VerifyPassword("secret password", tampered)
	if err != nil || ok {
		t.Errorf("tampered parameters: ok = %v, err = %v; want false", ok, err)
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	valid := hash("x", Params{Memory: 8192, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32})
	parts := strings.Split(valid, "$")
	salt, key := parts[4], parts[5]

	tests := map[string]string{
		"empty":             "",
		"plain text":        "hunter2",
		"other algorithm":   "$argon2i$v=19$m=8192,t=1,p=1$" + salt + "$" + key,
		"other version":     "$argon2id$v=16$m=8192,t=1,p=1$" + salt + "$" + key,
		"missing key":       "$argon2id$v=19$m=8192,t=1,p=1$" + salt,
		"zero time":         "$argon2id$v=19$m=8192,t=0,p=1$" + salt + "$" + key,
		"zero threads":      "$argon2id$v=19$m=8192,t=1,p=0$" + salt + "$" + key,
		"too many threads":  "$argon2id$v=19$m=8192,t=1,p=256$" + salt + "$" + key,
		"bad params":        "$argon2id$v=19$memory=8192$" + salt + "$" + key,
		"bad salt encoding": "$argon2id$v=19$m=8192,t=1,p=1$!!!$" + key,
		"empty key":         "$argon2id$v=19$m=8192,t=1,p=1$" + salt + "$",
	}
	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			ok, err := VerifyPassword("x", encoded)
			if ok || !errors.Is(err, ErrMalformedHash) {
				t.Errorf("ok = %v, err = %v; want false, ErrMalformedHash", ok, err)
			}
		})
	}
}

func TestDummyHashIsAValidHash(t *testing.T) {
	ok, err := VerifyPassword("anything", dummyHash())
	if err != nil || ok {
		t.Errorf("dummy hash: ok = %v, err = %v; want a valid hash no password matches", ok, err)
	}
	VerifyDummy("anything")
}

func TestNewSessionToken(t *testing.T) {
	token, hash := NewSessionToken()

	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token %q is not base64url: %v", token, err)
	}
	if len(raw) != TokenBytes {
		t.Errorf("token has %d bytes, want %d", len(raw), TokenBytes)
	}
	if !bytes.Equal(hash, HashSessionToken(token)) {
		t.Error("returned hash differs from HashSessionToken(token)")
	}
	if len(hash) != 32 {
		t.Errorf("hash has %d bytes, want 32 (SHA-256)", len(hash))
	}

	if other, _ := NewSessionToken(); other == token {
		t.Error("two tokens are equal")
	}
}
