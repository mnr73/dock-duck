// Package auth implements DockDuck's app login: password hashing, sessions and
// login rate limiting. The Docker socket is root-equivalent, so everything here
// errs on the side of refusing.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Policy bounds. The minimum is higher than for a typical web app on purpose.
const (
	MinPasswordLen = 10
	MaxPasswordLen = 256
)

// argon2id cost (OWASP-class). Variables so tests can use a cheap setting; Verify
// always uses the parameters stored in the hash itself.
var (
	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // KiB
	argonThreads uint8  = 2
)

const (
	saltLen = 16
	keyLen  = 32
)

// ErrPolicy is returned for passwords that are too short/long or blank.
var ErrPolicy = errors.New("password policy")

// CheckPolicy validates a new password.
func CheckPolicy(pw string) error {
	n := utf8.RuneCountInString(pw)
	switch {
	case strings.ContainsRune(pw, 0):
		return fmt.Errorf("%w: contains a NUL character", ErrPolicy)
	case strings.TrimSpace(pw) == "":
		return fmt.Errorf("%w: password is blank", ErrPolicy)
	case n < MinPasswordLen:
		return fmt.Errorf("%w: use at least %d characters", ErrPolicy, MinPasswordLen)
	case n > MaxPasswordLen:
		return fmt.Errorf("%w: use at most %d characters", ErrPolicy, MaxPasswordLen)
	}
	return nil
}

// Hash returns a PHC string: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<key>.
func Hash(pw string) (string, error) {
	m, t, p := argonMemory, argonTime, argonThreads
	if os.Getenv("DOCKDUCK_TEST_FAST_ARGON") == "1" && testing.Testing() {
		m, t, p = 64, 1, 1 // unit tests of other packages: the real cost would make them take minutes
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, t, m, p, keyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, m, t, p, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

// Verify checks pw against a PHC string in constant time. A malformed or
// out-of-bounds hash is an error (never a match): the file is read from disk, so a
// crafted `m=` must not be able to exhaust the server's memory.
func Verify(pw, phc string) (bool, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return false, errors.New("unsupported password hash")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errors.New("unsupported argon2 version")
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, errors.New("invalid argon2 parameters")
	}
	if m < 8 || m > 1<<20 || t < 1 || t > 20 || p < 1 || p > 32 {
		return false, errors.New("argon2 parameters out of range")
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false, errors.New("invalid salt")
	}
	want, err := enc.DecodeString(parts[5])
	if err != nil || len(want) < 16 || len(want) > 64 {
		return false, errors.New("invalid hash")
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
