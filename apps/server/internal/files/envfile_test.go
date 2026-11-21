package files

import (
	"slices"
	"testing"
)

func TestParseEnv(t *testing.T) {
	in := "# app\nAPP_NAME=Shiva\nexport DEBUG=true\n\nDB_PASSWORD=\"s3 cret # not a comment\"\nURL='http://x'  \nTRAIL=value # comment\nEMPTY=\nbad line\n1BAD=x\nMULTI=\"line one\nrest\"\nWIN=1\r\n"
	got := ParseEnv(in)
	kinds := []string{}
	for _, l := range got {
		kinds = append(kinds, l.Kind)
	}
	want := []string{"comment", "var", "var", "blank", "var", "var", "var", "var", "invalid", "invalid", "invalid", "invalid", "var"}
	if !slices.Equal(kinds, want) {
		t.Fatalf("kinds\n got  %v\n want %v", kinds, want)
	}
	byKey := map[string]EnvLine{}
	for _, l := range got {
		if l.Kind == "var" {
			byKey[l.Key] = l
		}
	}
	if v := byKey["DB_PASSWORD"]; v.Value != "s3 cret # not a comment" || v.Quote != `"` || !v.Secret || !v.Set {
		t.Errorf("quoted secret: %+v", v)
	}
	if v := byKey["URL"]; v.Value != "http://x" || v.Quote != "'" {
		t.Errorf("single quoted: %+v", v)
	}
	if v := byKey["TRAIL"]; v.Value != "value" {
		t.Errorf("trailing comment: %+v", v)
	}
	if v := byKey["EMPTY"]; v.Value != "" || v.Set {
		t.Errorf("empty: %+v", v)
	}
	if !byKey["DEBUG"].Export {
		t.Error("export prefix")
	}
	if v := byKey["WIN"]; v.Value != "1" {
		t.Errorf("CRLF: %+v", v)
	}
	if got := ParseEnv(""); got == nil || len(got) != 0 {
		t.Errorf("empty input must be an empty (non-nil) slice: %#v", got)
	}
}

func TestSecretDetectionAndMasking(t *testing.T) {
	for k, want := range map[string]bool{
		"DB_PASSWORD": true, "APP_KEY": true, "API_TOKEN": true, "AWS_SECRET_ACCESS_KEY": true, "SENTRY_LARAVEL_DSN": true,
		"APP_NAME": false, "DB_HOST": false, "PORT": false, "app_key": true, "MONKEY_PATCH": true, // keyword match is deliberately broad
	} {
		if IsSecretKey(k) != want {
			t.Errorf("IsSecretKey(%q) != %v", k, want)
		}
	}
	lines := Mask(ParseEnv("DB_HOST=db\nDB_PASSWORD=hunter2\nAPP_KEY=\nTOKEN=abc\n"))
	if lines[0].Value != "db" || lines[0].Masked {
		t.Errorf("non-secret untouched: %+v", lines[0])
	}
	if lines[1].Value != "" || !lines[1].Masked || !lines[1].Set {
		t.Errorf("secret masked but known to be set: %+v", lines[1])
	}
	if lines[2].Masked || lines[2].Set {
		t.Errorf("an empty secret has nothing to hide: %+v", lines[2])
	}
	if !lines[3].Masked || lines[3].Value != "" {
		t.Errorf("%+v", lines[3])
	}
	// the original slice is not modified
	orig := ParseEnv("DB_PASSWORD=x\n")
	_ = Mask(orig)
	if orig[0].Value != "x" {
		t.Error("Mask must copy")
	}
}

func TestKeys(t *testing.T) {
	if got := Keys("A=1\n# c\nB=2\nnot valid\n"); !slices.Equal(got, []string{"A", "B"}) {
		t.Errorf("%v", got)
	}
}
