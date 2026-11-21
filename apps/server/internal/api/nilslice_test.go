package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// The UI does `usage.services.length` and `usage.interpolated.length`: a JSON null
// there blanks the whole Environment tab. A project that does not use env_file: .env
// and has no ${VAR} is the common case.
func TestEnvUsageListsAreNeverNull(t *testing.T) {
	if got := interpolatedVars("no variables here"); got == nil {
		t.Error("interpolatedVars returned nil: marshals to null")
	}
	b, _ := json.Marshal(map[string]any{"interpolated": interpolatedVars("plain")})
	if strings.Contains(string(b), "null") {
		t.Errorf("%s", b)
	}
}
