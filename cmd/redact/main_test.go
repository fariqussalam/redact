package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIHelpUsesParserHelp(t *testing.T) {
	bin := build(t)
	cmd := exec.Command(bin, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	s := string(out)
	for _, want := range []string{"Usage: redact <command> [flags]", "add-name <name>", "--config=STRING"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in\n%s", want, s)
		}
	}
}

func TestCLIStreamWithExplicitConfig(t *testing.T) {
	bin := build(t)
	cfg := filepath.Join(t.TempDir(), "redact.yaml")
	if err := os.WriteFile(cfg, []byte("use_defaults: true\nmask: \"[X]\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "--config", cfg)
	cmd.Stdin = strings.NewReader("Authorization: Bearer abc\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if string(out) != "Authorization: [X]\n" {
		t.Fatalf("out=%q", out)
	}
}

func TestCLIMissingExplicitConfigErrors(t *testing.T) {
	bin := build(t)
	cmd := exec.Command(bin, "--config", filepath.Join(t.TempDir(), "missing.yaml"))
	cmd.Stdin = strings.NewReader("Authorization: Bearer abc\n")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error, got %q", out)
	}
	if !strings.Contains(string(out), "config file not found") {
		t.Fatalf("out=%q", out)
	}
}

func TestCLIAddListAndRemoveName(t *testing.T) {
	bin := build(t)
	cfg := filepath.Join(t.TempDir(), "redact.yaml")

	out, err := exec.Command(bin, "--config", cfg, "add-name", "X-Company-Token").CombinedOutput()
	if err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	b, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "x-company-token") {
		t.Fatalf("config=\n%s", b)
	}

	out, err = exec.Command(bin, "--config", cfg, "list", "--user").CombinedOutput()
	if err != nil {
		t.Fatalf("list: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "x-company-token") || strings.Contains(string(out), "authorization") {
		t.Fatalf("list --user output=\n%s", out)
	}

	out, err = exec.Command(bin, "--config", cfg, "remove-name", "x-company-token").CombinedOutput()
	if err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	b, err = os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "x-company-token") {
		t.Fatalf("config=\n%s", b)
	}
}

func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "redact")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}
