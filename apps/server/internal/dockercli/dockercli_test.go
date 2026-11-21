package dockercli

import (
	"errors"
	"testing"
)

func TestGuard(t *testing.T) {
	ok := [][]string{
		{"context", "ls", "--format", "json"},
		{"--context", "context", "ps"},
		{"context", "inspect", "default"},
		{"compose", "ls", "-a"},
	}
	bad := [][]string{
		{"context", "create", "x"},
		{"--context", "a", "context", "rm", "b"},
		{"context", "use", "x"},
		{"context", "update", "x"},
		{"context", "import", "x", "f"},
		{"context"},
	}
	for _, a := range ok {
		if err := guard(a); err != nil {
			t.Errorf("%v should be allowed: %v", a, err)
		}
	}
	for _, a := range bad {
		if err := guard(a); !errors.Is(err, ErrForbidden) {
			t.Errorf("%v should be forbidden, got %v", a, err)
		}
	}
}

func TestValidName(t *testing.T) {
	for s, want := range map[string]bool{"default": true, "my-ctx_1.x": true, "": false, "-x": false, "a b": false, "a;b": false, "1000": true} {
		if ValidName(s) != want {
			t.Errorf("ValidName(%q) != %v", s, want)
		}
	}
}

func TestDecodeLines(t *testing.T) {
	type T struct{ N string }
	a, err := DecodeLines[T]([]byte("{\"N\":\"a\"}\n{\"N\":\"b\"}\n"))
	if err != nil || len(a) != 2 || a[1].N != "b" {
		t.Fatalf("ndjson: %v %v", a, err)
	}
	b, err := DecodeLines[T]([]byte(`[{"N":"x"}]`))
	if err != nil || len(b) != 1 {
		t.Fatalf("array: %v %v", b, err)
	}
	if c, _ := DecodeLines[T](nil); c != nil {
		t.Fatal("empty")
	}
}
