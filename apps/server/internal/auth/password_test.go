package auth

import (
	"strings"
	"testing"
)

func cheap(t *testing.T) {
	t.Helper()
	m, tt, p := argonMemory, argonTime, argonThreads
	argonMemory, argonTime, argonThreads = 64, 1, 1
	t.Cleanup(func() { argonMemory, argonTime, argonThreads = m, tt, p })
}

func TestHashVerify(t *testing.T) {
	cheap(t)
	h, err := Hash("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Errorf("PHC format: %s", h)
	}
	if strings.Contains(h, "correct") {
		t.Error("the hash must not contain the password")
	}
	if ok, err := Verify("correct horse battery", h); !ok || err != nil {
		t.Errorf("right password: %v %v", ok, err)
	}
	for _, bad := range []string{"", "correct horse batterY", "correct horse battery ", "x"} {
		if ok, _ := Verify(bad, h); ok {
			t.Errorf("%q must not verify", bad)
		}
	}
	h2, _ := Hash("correct horse battery")
	if h == h2 {
		t.Error("every hash has its own salt")
	}
}

// The default cost must be the documented one; a verify of a real default-cost hash works.
func TestDefaultCostIsStoredInTheHash(t *testing.T) {
	h, err := Hash("a long enough password")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h, "m=65536,t=3,p=2") {
		t.Errorf("default parameters: %s", h)
	}
	if ok, _ := Verify("a long enough password", h); !ok {
		t.Error("verify with the parameters stored in the hash")
	}
}

func TestVerifyRejectsMalformedAndHostileHashes(t *testing.T) {
	cheap(t)
	good, _ := Hash("whatever password")
	salt := strings.Split(good, "$")[4]
	key := strings.Split(good, "$")[5]
	for name, h := range map[string]string{
		"empty":         "",
		"plain text":    "hunter2hunter2",
		"wrong algo":    "$argon2i$v=19$m=64,t=1,p=1$" + salt + "$" + key,
		"bad version":   "$argon2id$v=16$m=64,t=1,p=1$" + salt + "$" + key,
		"huge memory":   "$argon2id$v=19$m=4194304,t=1,p=1$" + salt + "$" + key,
		"zero time":     "$argon2id$v=19$m=64,t=0,p=1$" + salt + "$" + key,
		"huge time":     "$argon2id$v=19$m=64,t=999,p=1$" + salt + "$" + key,
		"zero threads":  "$argon2id$v=19$m=64,t=1,p=0$" + salt + "$" + key,
		"short salt":    "$argon2id$v=19$m=64,t=1,p=1$AAAA$" + key,
		"short key":     "$argon2id$v=19$m=64,t=1,p=1$" + salt + "$AAAA",
		"bad base64":    "$argon2id$v=19$m=64,t=1,p=1$!!!$" + key,
		"missing parts": "$argon2id$v=19$m=64,t=1,p=1$" + salt,
		"extra parts":   good + "$x",
	} {
		if ok, err := Verify("whatever password", h); ok || err == nil {
			t.Errorf("%s: must be an error, got ok=%v err=%v", name, ok, err)
		}
	}
}

func TestPolicy(t *testing.T) {
	ok := []string{"0123456789", "correct horse battery", "пароль-длинный-123", strings.Repeat("a", MaxPasswordLen)}
	for _, p := range ok {
		if err := CheckPolicy(p); err != nil {
			t.Errorf("%q should pass: %v", p, err)
		}
	}
	bad := []string{"", "short", "123456789", "          ", "\t\t\t\t\t\t\t\t\t\t\t", "abc\x00defghijk", strings.Repeat("a", MaxPasswordLen+1)}
	for _, p := range bad {
		if err := CheckPolicy(p); err == nil {
			t.Errorf("%q must be refused", p)
		}
	}
}
