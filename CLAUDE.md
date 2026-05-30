# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Build
go build ./...

# Run all tests (Docker required for integration tests)
go test ./... -v -timeout 300s

# Run unit tests only (no Docker)
SKIP_MAIL=1 go test ./cmd/... -v

# Run a single test
go test ./cmd/... -run TestSendCommand -v
go test ./cmd/... -run TestMailDocker -v -timeout 300s

# Lint
./lintGo.sh
# or directly:
golangci-lint run ./... --timeout=5m
```

## Architecture

The CLI is built with **cobra** (commands) + **viper** (config).
All mail operations delegate to `github.com/tommi2day/gomodules/maillib`.

### Config precedence (lowest → highest)
1. YAML config file (`~/etc/mailcli.yaml` or `./mailcli.yaml`, or `--config` flag)
2. Environment variables with prefix `MAILCLI_` (dots replaced by underscores, e.g. `MAILCLI_SMTP_SERVER`)
3. CLI flags

### Dotted flag names
Flags use dots to match YAML nested keys and env var mapping:
- `--smtp.server` → yaml `smtp.server` → env `MAILCLI_SMTP_SERVER`
- `--imap.server` → yaml `imap.server` → env `MAILCLI_IMAP_SERVER`

All subcommand flags are bound to viper via `viper.BindPFlags(cmd.Flags())` so `viper.GetString("smtp.server")` automatically resolves the right source.

### Command structure
- `cmd/rootcmd.go` — `RootCmd`, `Execute()`, `initConfig()`, viper/cobra wiring, `hideGlobalFlags()` helper
- `cmd/send.go` — `send` subcommand, wraps `maillib.SendMailConfigType.SendMail()`
- `cmd/imap.go` — `imap` parent + `list/status/read/search/delete` subcommands, wraps `maillib.ImapType`
- `cmd/version.go` — `version` subcommand; `Version`, `Commit`, `Date` vars injected at build time via ldflags

### Test layout
- `test/testinit.go` — `InitTestDirs()` changes the working directory to `test/`; auto-called by `init()` so it runs for all tests in the `cmd` package that import `github.com/tommi2day/mailcli/test`
- `test/mailcli.yaml` — test config pointing to `127.0.0.1` on docker ports
- `test/no_config.yaml` — intentionally empty config for unit tests that need no fallback values
- `test/docker/mail/` — docker-mailserver config files (accounts, SSL certs)
- `cmd/mail_docker_test.go` — `prepareMailContainer()` starts `docker-mailserver:15.1.0`; set `SKIP_MAIL=1` to bypass
- `cmd/mail_test.go` — `TestMailDocker` integration tests (send via SMTP, read/delete via IMAP)

### Critical: cobra flag state in tests
Cobra does **not** reset flag values between `cmd.Execute()` calls when the same command object is reused. Because flags are bound to package-level variables, values from one subtest leak into the next. Additionally, `pflag.Flag.Changed` must be reset to `false` so viper falls back to yaml/env instead of the stale flag value.

Call `resetSmtpState()` / `resetImapState()` at the start of every send/imap subtest. These helpers reset both the Go variables and `f.Changed = false` on all flags.

