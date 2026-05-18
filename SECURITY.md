# Security Policy

`redact` is a local best-effort text redactor. It is designed to reduce accidental secret sharing in logs and command output, but it cannot guarantee that every secret format will be detected.

## Reporting a vulnerability

If you find a redaction bypass, crash, unsafe file permission, or other security issue, please open a private security advisory on GitHub if available. If not, contact the maintainer directly before publishing details.

Please include:

- The `redact` version or commit.
- The input that was not handled correctly, with real secrets replaced by fake examples.
- Your OS and shell, if relevant.
- Expected and actual output.

## Handling secrets safely

When reporting issues, do not include real API keys, tokens, passwords, private keys, or production logs. Use synthetic examples.

## Supported versions

Until the project reaches a stable v1 release, security fixes are provided on the latest main branch and latest tagged release only.
