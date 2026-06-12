# redact

Local streaming text redactor for developers. Pipe command output through `redact` before sharing logs with people or AI tools.

`redact` stores rule names like `authorization` and `access_token`, not your secret values. It makes no network requests.

```sh
curl -v https://api.example.com \
  -H "Authorization: Bearer sk_live_abc" \
  -H "x-api-key: secret123" 2>&1 | redact
```

Output:

```text
Authorization: ***
x-api-key: ***
```

## Install

### Recommended — one-liner (no Go required)

```sh
curl -fsSL https://raw.githubusercontent.com/fariqussalam/redact/master/install.sh | sh
```

Detects your OS and architecture, downloads the latest release, verifies its checksum, and installs `redact` to `/usr/local/bin`.

### Via Go (requires Go)

```sh
go install github.com/fariqussalam/redact/cmd/redact@latest
```

Make sure `~/go/bin` is on your `PATH`. Add this to `~/.zshrc` or `~/.bashrc` if not already present:

```sh
export PATH="$HOME/go/bin:$PATH"
```

## Usage

```sh
some_command 2>&1 | redact
redact < input.log > redacted.log
redact --buffered < large.log > redacted.log
redact clip           # redact clipboard and copy result back
redact clip --print   # also print redacted clipboard contents

redact add-name x-company-token
redact remove-name x-company-token
redact add-pattern 'token'
redact list
redact list --user
redact list --defaults
redact config-path
```

If stdin is a terminal, `redact` prints help. If stdin is piped, it redacts stdin to stdout.

## Config

Print the config path:

```sh
redact config-path
```

Config is created when you first add a rule.

```yaml
use_defaults: true
mask: "***"
names: []
patterns: []
```

Rules are matched case-insensitively. `names` apply to both field-like syntax (`password=...`, `Authorization: ...`) and URL query params (`?password=...`). `patterns` are regex substring matches against detected field or URL parameter names only.

Config files are stored with owner-only permissions when `redact` writes them.

## Security model

`redact` is a local, best-effort filter for same-line text logs.

It does:

- Read from stdin or the clipboard.
- Write redacted text to stdout or back to the clipboard.
- Load local config rules.

It does not:

- Make network requests.
- Store secret values.
- Learn from your logs.
- Guarantee every possible secret format is detected.

Always review redacted output before sharing sensitive logs publicly.

## Supported formats

Current v1 scope supports common same-line text forms:

- `field: value`
- `field=value`
- JSON-ish `"field": "value"`
- URL query params
- Authorization headers as whole values
## Development

```sh
go test ./...
go test -race ./...
go vet ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

MIT
