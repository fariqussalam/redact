# redact

CLI to redact / strip / mask sensitive data from your text.

`redact` stores rule names like `authorization` and `access_token` and apply masking to its values.

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

### one-liner

```sh
curl -fsSL https://raw.githubusercontent.com/fariqussalam/redact/master/install.sh | sh
```

### Via Go

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

Always review redacted output before sharing sensitive logs publicly.

## Supported formats

Current v1 scope supports common same-line text forms:

- `field: value`
- `field=value`
- JSON-ish `"field": "value"`
- URL query params
- Authorization headers as whole values
## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md).

## License

MIT
