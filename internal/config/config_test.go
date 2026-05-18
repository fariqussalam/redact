package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingMaskDefaultsButEmptyMaskErrors(t *testing.T) {
	cfg, _, err := Load(writeTemp(t, "use_defaults: true\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if got := EffectiveRules(cfg).Mask; got != DefaultMask {
		t.Fatalf("mask=%q", got)
	}

	_, _, err = Load(writeTemp(t, "mask: \"\"\n"), true)
	if err == nil || !strings.Contains(err.Error(), "mask cannot be empty") {
		t.Fatalf("expected empty mask error, got %v", err)
	}
}

func TestExplicitMissingConfigErrors(t *testing.T) {
	_, _, err := Load(filepath.Join(t.TempDir(), "missing.yaml"), true)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateRejectsInvalidManualNames(t *testing.T) {
	_, _, err := Load(writeTemp(t, "names:\n  - 'bad:name'\n"), true)
	if err == nil || !strings.Contains(err.Error(), "invalid name") {
		t.Fatalf("expected invalid name error, got %v", err)
	}
}

func TestValidateRejectsInvalidRegex(t *testing.T) {
	_, _, err := Load(writeTemp(t, "patterns:\n  - '['\n"), true)
	if err == nil || !strings.Contains(err.Error(), "invalid regex") {
		t.Fatalf("expected invalid regex error, got %v", err)
	}
}

func TestEffectiveRules(t *testing.T) {
	useDefaults := false
	mask := "[redacted]"
	cfg := Config{
		UseDefaults: &useDefaults,
		Mask:        &mask,
		Names:       []string{"Shared_Token"},
		Patterns:    []string{"shared"},
	}

	e := EffectiveRules(cfg)
	if e.UseDefaults {
		t.Fatal("UseDefaults=true, want false")
	}
	if e.Mask != mask {
		t.Fatalf("Mask=%q, want %q", e.Mask, mask)
	}
	if !ContainsRule(e.Fields, "shared_token") || !ContainsRule(e.URLParams, "shared_token") {
		t.Fatalf("Names should apply to fields and URL params: %+v", e)
	}
	if !ContainsRule(e.FieldPatterns, "shared") || !ContainsRule(e.URLParamPatterns, "shared") {
		t.Fatalf("Patterns should apply to fields and URL params: %+v", e)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "lowercase trim", in: " X-Token ", want: "x-token"},
		{name: "empty", in: " ", wantErr: true},
		{name: "invalid separator", in: "bad:name", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeName(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMutatePreservesCommentsAndUnknowns(t *testing.T) {
	path := writeTemp(t, "# top\nuse_defaults: true\nunknown: yes\nnames:\n  # company\n  - x-token\n")
	changed, err := AddListValue(path, "names", "new-token")
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"# top", "unknown: yes", "# company", "- x-token", "- new-token"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func writeTemp(t *testing.T, s string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}
