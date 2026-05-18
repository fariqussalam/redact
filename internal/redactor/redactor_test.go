package redactor

import (
	"strings"
	"testing"

	"github.com/fariq/redact/internal/config"
)

func TestRedactLine(t *testing.T) {
	r := newTestRedactor(t, config.Effective{
		Mask: "***",
		Fields: []config.Rule{
			{Value: "authorization", Source: "user"},
			{Value: "x-api-key", Source: "user"},
			{Value: "client_secret", Source: "user"},
			{Value: "password", Source: "user"},
		},
		URLParams: []config.Rule{
			{Value: "access_token", Source: "user"},
		},
		FieldPatterns: []config.Rule{
			{Value: "token", Source: "user"},
		},
		URLParamPatterns: []config.Rule{
			{Value: "api", Source: "user"},
		},
	})

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "authorization header",
			in:   "Authorization: Bearer abc.def\n",
			want: "Authorization: ***\n",
		},
		{
			name: "crlf header",
			in:   "> x-api-key: secret\r\n",
			want: "> x-api-key: ***\r\n",
		},
		{
			name: "equals field stops before next key",
			in:   "client_secret=abc foo=bar",
			want: "client_secret=*** foo=bar",
		},
		{
			name: "json string value",
			in:   `{"password":"hunter2","ok":true}`,
			want: `{"password":"***","ok":true}`,
		},
		{
			name: "json escaped quote",
			in:   `{"token":"abc\"def","ok":true}`,
			want: `{"token":"***","ok":true}`,
		},
		{
			name: "url exact and pattern params",
			in:   "https://x.com?a=1&access_token=abc%2Fdef&api_key=xyz#frag",
			want: "https://x.com?a=1&access_token=***&api_key=***#frag",
		},
		{
			name: "non matching prose",
			in:   "the secret sauce is garlic",
			want: "the secret sauce is garlic",
		},
		{
			name: "field name boundary",
			in:   "notAuthorization: Bearer abc",
			want: "notAuthorization: Bearer abc",
		},
		{
			name: "curl quoted header and url",
			in:   "curl -H \"authorization: Bearer def\" \"https://x.com?access_token=abc\"",
			want: "curl -H \"authorization: ***\" \"https://x.com?access_token=***\"",
		},
		{
			name: "logfmt style",
			in:   "2026 INFO client_secret=abc status=200 request_id=123",
			want: "2026 INFO client_secret=*** status=200 request_id=123",
		},
		{
			name: "nested json header and url string",
			in:   `{"headers":{"Authorization":"Bearer abc.def"},"url":"https://x.com?access_token=abc"}`,
			want: `{"headers":{"Authorization":"***"},"url":"https://x.com?access_token=***"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(r.RedactLine(nil, []byte(tt.in)))
			if got != tt.want {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestCustomMaskEscaping(t *testing.T) {
	r := newTestRedactor(t, config.Effective{
		Mask:      `*"x y`,
		Fields:    []config.Rule{{Value: "password", Source: "user"}},
		URLParams: []config.Rule{{Value: "access_token", Source: "user"}},
	})

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain field uses raw mask",
			in:   "password=secret",
			want: `password=*"x y`,
		},
		{
			name: "json string uses json escaped mask",
			in:   `{"password":"secret"}`,
			want: `{"password":"*\"x y"}`,
		},
		{
			name: "url param uses query escaped mask",
			in:   "https://x.com?access_token=secret",
			want: "https://x.com?access_token=*%22x+y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(r.RedactLine(nil, []byte(tt.in)))
			if got != tt.want {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestInvalidRegexPatternErrors(t *testing.T) {
	_, err := New(config.Effective{
		Mask:          "***",
		FieldPatterns: []config.Rule{{Value: "[", Source: "user"}},
	})
	if err == nil || !strings.Contains(err.Error(), "missing closing ]") {
		t.Fatalf("expected regex compile error, got %v", err)
	}
}

func TestRedactLineReusesDestination(t *testing.T) {
	r := newTestRedactor(t, config.Effective{
		Mask:   "***",
		Fields: []config.Rule{{Value: "password", Source: "user"}},
	})

	dst := make([]byte, 0, 64)
	got := r.RedactLine(dst, []byte("password=secret"))
	if string(got) != "password=***" {
		t.Fatalf("got %q", got)
	}
	if cap(got) != cap(dst) {
		t.Fatalf("expected destination buffer reuse")
	}
}

func FuzzRedactLineDoesNotPanic(f *testing.F) {
	r := mustRedactor(f, config.Effective{
		Mask:      "***",
		Fields:    []config.Rule{{Value: "password", Source: "user"}},
		URLParams: []config.Rule{{Value: "access_token", Source: "user"}},
	})

	for _, seed := range []string{
		"password=secret",
		`{"password":"secret"}`,
		"https://x.com?access_token=secret",
		"unterminated \"password: secret",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = r.RedactLine(nil, []byte(input))
	})
}

func BenchmarkRedactLine(b *testing.B) {
	r := mustRedactor(b, config.Effective{
		Mask:          "***",
		Fields:        []config.Rule{{Value: "authorization", Source: "user"}, {Value: "password", Source: "user"}},
		URLParams:     []config.Rule{{Value: "access_token", Source: "user"}},
		FieldPatterns: []config.Rule{{Value: "token", Source: "user"}},
	})
	line := []byte(`2026 INFO authorization="Bearer abc" password=hunter2 url="https://x.com?access_token=secret" status=200`)
	dst := make([]byte, 0, len(line))

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst = r.RedactLine(dst[:0], line)
	}
}

func newTestRedactor(t *testing.T, e config.Effective) *Redactor {
	t.Helper()
	return mustRedactor(t, e)
}

func mustRedactor(tb testing.TB, e config.Effective) *Redactor {
	tb.Helper()
	r, err := New(e)
	if err != nil {
		tb.Fatal(err)
	}
	return r
}
