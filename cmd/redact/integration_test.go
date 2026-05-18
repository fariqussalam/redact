package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIIntegrationSuite(t *testing.T) {
	bin := build(t)
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")

	writeFile(t, cfg, "use_defaults: true\nmask: \"[X]\"\n")

	tests := []struct {
		name     string
		args     []string
		stdin    string
		wantOut  string
		wantErr  string
		wantFail bool
	}{
		{
			name:    "help",
			args:    []string{"--help"},
			wantOut: "Usage: redact <command> [flags]",
		},
		{
			name:    "stream default rules",
			stdin:   "Authorization: Bearer abc\n",
			wantOut: "Authorization: ***\n",
		},
		{
			name:    "stream explicit config",
			args:    []string{"--config", cfg},
			stdin:   "x-api-key: secret\n",
			wantOut: "x-api-key: [X]\n",
		},
		{
			name:    "buffered stream explicit config",
			args:    []string{"--config", cfg, "--buffered"},
			stdin:   "x-api-key: secret\n",
			wantOut: "x-api-key: [X]\n",
		},
		{
			name:    "stream without trailing newline",
			stdin:   "password=hunter2",
			wantOut: "password=***",
		},
		{
			name:     "missing explicit config",
			args:     []string{"--config", filepath.Join(dir, "missing.yaml")},
			stdin:    "Authorization: Bearer abc\n",
			wantErr:  "config file not found",
			wantFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runCLI(t, bin, tt.stdin, tt.args...)
			if tt.wantFail {
				if err == nil {
					t.Fatalf("expected command to fail, output:\n%s", out)
				}
				if !strings.Contains(out, tt.wantErr) {
					t.Fatalf("output missing %q:\n%s", tt.wantErr, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("%v\n%s", err, out)
			}
			if tt.wantOut != "" && !strings.Contains(out, tt.wantOut) {
				t.Fatalf("output missing %q:\n%s", tt.wantOut, out)
			}
		})
	}
}

func TestCLIIntegrationConfigMutationFlow(t *testing.T) {
	bin := build(t)
	cfg := filepath.Join(t.TempDir(), "config.yaml")

	out, err := runCLI(t, bin, "", "--config", cfg, "add-name", "X-Company-Token")
	if err != nil {
		t.Fatalf("add-name: %v\n%s", err, out)
	}
	if !strings.Contains(out, "added: x-company-token") {
		t.Fatalf("unexpected add output:\n%s", out)
	}

	out, err = runCLI(t, bin, "", "--config", cfg, "list", "--user")
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if !strings.Contains(out, "x-company-token") {
		t.Fatalf("list output missing added field:\n%s", out)
	}

	out, err = runCLI(t, bin, "", "--config", cfg, "remove-name", "x-company-token")
	if err != nil {
		t.Fatalf("remove-name: %v\n%s", err, out)
	}
	if !strings.Contains(out, "removed: x-company-token") {
		t.Fatalf("unexpected remove output:\n%s", out)
	}

	out, err = runCLI(t, bin, "", "--config", cfg, "list", "--user")
	if err != nil {
		t.Fatalf("list after remove: %v\n%s", err, out)
	}
	if strings.Contains(out, "x-company-token") {
		t.Fatalf("removed field still listed:\n%s", out)
	}

	out, err = runCLI(t, bin, "", "--config", cfg, "add-pattern", "token")
	if err != nil {
		t.Fatalf("add-pattern: %v\n%s", err, out)
	}
	out, err = runCLI(t, bin, "x-company-token=abc https://x.com?session_token=def", "--config", cfg)
	if err != nil {
		t.Fatalf("redact pattern: %v\n%s", err, out)
	}
	if want := "x-company-token=*** https://x.com?session_token=***"; out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func runCLI(t *testing.T, bin, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
