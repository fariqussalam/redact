package cli

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/fariqussalam/redact/internal/config"
	"github.com/fariqussalam/redact/internal/redactor"
)

func TestFirstCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "empty", args: nil, want: ""},
		{name: "command only", args: []string{"list"}, want: "list"},
		{name: "long config value", args: []string{"--config", "cfg.yaml", "list"}, want: "list"},
		{name: "long config equals", args: []string{"--config=cfg.yaml", "list"}, want: "list"},
		{name: "skips flags", args: []string{"--help"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstCommand(tt.args); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantConfig   string
		wantBuffered bool
		wantHelp     bool
	}{
		{name: "none"},
		{name: "config value", args: []string{"--config", "cfg.yaml"}, wantConfig: "cfg.yaml"},
		{name: "config equals", args: []string{"--config=cfg.yaml"}, wantConfig: "cfg.yaml"},
		{name: "buffered", args: []string{"--buffered"}, wantBuffered: true},
		{name: "long help", args: []string{"--help"}, wantHelp: true},
		{name: "short help", args: []string{"-h"}, wantHelp: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := flagConfig(tt.args); got != tt.wantConfig {
				t.Fatalf("flagConfig got %q, want %q", got, tt.wantConfig)
			}
			if got := flagBuffered(tt.args); got != tt.wantBuffered {
				t.Fatalf("flagBuffered got %v, want %v", got, tt.wantBuffered)
			}
			if got := hasHelp(tt.args); got != tt.wantHelp {
				t.Fatalf("hasHelp got %v, want %v", got, tt.wantHelp)
			}
		})
	}
}

func TestStream(t *testing.T) {
	r := newTestRedactor(t)
	var out strings.Builder

	if err := stream(r, strings.NewReader("password=secret\n"), &out, true); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "password=***\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStreamPropagatesWriteErrors(t *testing.T) {
	r := newTestRedactor(t)
	err := stream(r, strings.NewReader("password=secret\n"), errWriter{}, false)
	if err == nil || !strings.Contains(err.Error(), "redacted output") {
		t.Fatalf("expected output error, got %v", err)
	}
}

func newTestRedactor(t *testing.T) *redactor.Redactor {
	t.Helper()
	r, err := redactor.New(config.Effective{
		Mask:   config.DefaultMask,
		Fields: []config.Rule{{Value: "password", Source: "user"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

var _ io.Writer = errWriter{}
