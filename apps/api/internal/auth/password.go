// Package auth hashes passwords and creates session tokens.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Params are Argon2id cost parameters. Memory is in KiB.
type Params struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	SaltLen uint32
	KeyLen  uint32
}

// DefaultParams is the OWASP minimum for Argon2id: 19 MiB, 2 iterations,
// 1 degree of parallelism (tech spec §5.5).
var DefaultParams = Params{Memory: 19456, Time: 2, Threads: 1, SaltLen: 16, KeyLen: 32}

// ErrMalformedHash means a stored hash is not a PHC string this package
// can verify.
var ErrMalformedHash = errors.New("malformed password hash")

var b64 = base64.RawStdEncoding

// HashPassword hashes password with DefaultParams and returns a PHC string:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>. The parameters travel with
// the hash, so they can be raised later without invalidating old hashes.
func HashPassword(password string) string {
	return hash(password, DefaultParams)
}

func hash(password string, p Params) string {
	salt := make([]byte, p.SaltLen)
	_, _ = rand.Read(salt) // never fails; see crypto/rand.Read
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key))
}

// VerifyPassword reports whether password matches encoded, using the
// parameters stored in encoded. The comparison takes constant time.
func VerifyPassword(password, encoded string) (bool, error) {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1, nil
}

var dummyHash = sync.OnceValue(func() string {
	return HashPassword(rand.Text())
})

// VerifyDummy spends the same time as VerifyPassword and discards the
// result. Login calls it for unknown emails, so response time does not
// reveal which emails have an account.
func VerifyDummy(password string) {
	_, _ = VerifyPassword(password, dummyHash())
}

func decode(encoded string) (p Params, salt, key []byte, err error) {
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, key
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Params{}, nil, nil, ErrMalformedHash
	}

	var threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &threads); err != nil {
		return Params{}, nil, nil, ErrMalformedHash
	}
	// argon2.IDKey panics on zero time or threads.
	if p.Memory == 0 || p.Time == 0 || threads == 0 || threads > 255 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	p.Threads = uint8(threads)

	if salt, err = b64.DecodeString(parts[4]); err != nil || len(salt) == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	if key, err = b64.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return Params{}, nil, nil, ErrMalformedHash
	}
	return p, salt, key, nil
}
