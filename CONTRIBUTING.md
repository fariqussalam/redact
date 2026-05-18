# Contributing

Thanks for helping improve `redact`.

## Development

Requirements:

- Go 1.24+

Common commands:

```sh
go test ./...
go test -race ./...
go vet ./...
go mod tidy
```

Run the CLI locally:

```sh
go run ./cmd/redact < input.log
```

## Redaction changes

For any change that modifies what gets redacted:

- Add focused tests with fake secrets only.
- Keep matching explainable and bounded.
- Avoid storing or learning secret values.
- Preserve streaming behavior.
- Document new supported formats in `README.md`.

## Pull requests

Please keep PRs small and include:

- What changed.
- Why it changed.
- Any edge cases considered.
- Test results.
