package files

import (
	"regexp"
	"strings"
)

// EnvLine is one line of a .env file, structurally (comments/blank lines kept so
// the UI can save a table edit without losing them).
type EnvLine struct {
	Kind   string `json:"kind"`            // "var" | "comment" | "blank" | "invalid"
	Key    string `json:"key,omitempty"`   // var
	Value  string `json:"value,omitempty"` // var; real value only when revealed
	Quote  string `json:"quote,omitempty"` // "", `"` or `'`
	Export bool   `json:"export,omitempty"`
	Raw    string `json:"raw,omitempty"` // comment / invalid text
	Secret bool   `json:"secret,omitempty"`
	Masked bool   `json:"masked,omitempty"` // Value was replaced by a placeholder
	Set    bool   `json:"set,omitempty"`    // var has a non-empty value (meaningful when masked)
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.\-]*$`)

var secretWords = []string{"PASSWORD", "PASSWD", "SECRET", "TOKEN", "KEY", "CREDENTIAL", "PRIVATE", "AUTH", "DSN", "SALT", "PASS"}

// IsSecretKey reports whether the key name suggests a secret value.
func IsSecretKey(k string) bool {
	u := strings.ToUpper(k)
	for _, w := range secretWords {
		if strings.Contains(u, w) {
			return true
		}
	}
	return false
}

// ParseEnv splits content into lines. It understands `KEY=value`,
// `export KEY=value`, single/double quoted values and trailing ` # comments` on
// unquoted values. Multi-line quoted values are kept as `invalid` raw lines so
// they round-trip untouched.
func ParseEnv(content string) []EnvLine {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []EnvLine{}
	}
	var out []EnvLine
	for _, raw := range strings.Split(content, "\n") {
		t := strings.TrimSpace(raw)
		switch {
		case t == "":
			out = append(out, EnvLine{Kind: "blank"})
			continue
		case strings.HasPrefix(t, "#"):
			out = append(out, EnvLine{Kind: "comment", Raw: raw})
			continue
		}
		l := EnvLine{Kind: "var"}
		body := t
		if strings.HasPrefix(body, "export ") {
			l.Export, body = true, strings.TrimSpace(body[len("export "):])
		}
		k, v, ok := strings.Cut(body, "=")
		k = strings.TrimSpace(k)
		if !ok || !envKey.MatchString(k) {
			out = append(out, EnvLine{Kind: "invalid", Raw: raw})
			continue
		}
		l.Key = k
		v = strings.TrimSpace(v)
		switch {
		case len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && strings.HasSuffix(v, string(v[0])):
			l.Quote, v = string(v[0]), v[1:len(v)-1]
		case len(v) >= 1 && (v[0] == '"' || v[0] == '\''):
			// opening quote without a closing one on this line: multi-line value
			out = append(out, EnvLine{Kind: "invalid", Raw: raw})
			continue
		default:
			if i := strings.Index(v, " #"); i >= 0 {
				v = strings.TrimSpace(v[:i])
			}
		}
		l.Value = v
		l.Secret = IsSecretKey(k)
		l.Set = v != ""
		out = append(out, l)
	}
	return out
}

// Mask replaces the values of secret variables with a placeholder. `Set` tells
// the UI whether a value exists without sending it.
func Mask(lines []EnvLine) []EnvLine {
	out := make([]EnvLine, len(lines))
	copy(out, lines)
	for i := range out {
		if out[i].Kind == "var" && out[i].Secret && out[i].Value != "" {
			out[i].Value, out[i].Masked = "", true
		}
	}
	return out
}

// Keys returns the variable names defined in content.
func Keys(content string) []string {
	var ks []string
	for _, l := range ParseEnv(content) {
		if l.Kind == "var" {
			ks = append(ks, l.Key)
		}
	}
	return ks
}
