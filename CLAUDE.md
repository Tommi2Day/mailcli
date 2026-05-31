# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Build
go build ./...

# Run
go run main.go <command> [flags]

# Test (all, skipping Docker integration tests)
SKIP_MAIL=1 go test ./...

# Test a single test function
SKIP_MAIL=1 go test ./cmd/... -run TestSendCommand -v

# Run Docker integration tests (requires Docker)
go test ./cmd/... -v -run TestMailDocker

# Lint
golangci-lint run ./...

# Format
goimports -w ./...

# Release build (via goreleaser)
goreleaser build --snapshot --clean
```

## Architecture

`main.go` is a one-liner that calls `cmd.Execute()`. All logic lives in `cmd/`.

**Config resolution order** (lowest → highest priority):
1. YAML file auto-discovered from `./mailcli.yaml`, `~/.config/mailcli.yaml`, `~/etc/mailcli.yaml`, `/etc/mailcli.yaml`, or `--config` flag
2. Environment variables with `MAILCLI_` prefix, dots replaced by underscores (e.g. `MAILCLI_SMTP_SERVER`)
3. CLI flags

`rootcmd.go` wires Cobra + Viper: `initConfig()` is called via `cobra.OnInitialize` on every command invocation. `viper.BindPFlags` is called for each command's flag set so that flags, env vars, and config file keys all resolve through viper.

**Command files:**
- `send.go` — SMTP send; builds `SendMailConfigType` and `MailType` from viper, calls `maillib.SendMail`
- `imap.go` — IMAP subcommands (list/status/read/search/delete); builds `ImapType` from viper, then calls maillib methods
- `config.go` — `config show` and `config save` subcommands; reads `viper.AllSettings()` and marshals to YAML
- `sign.go` — standalone `sign` and `verify` commands; thin wrappers around `maillib.SignMailContent` / `maillib.VerifyMailSignature`
- `version.go` — prints version string; `Version`, `Commit`, `Date` are injected at build time via `-ldflags`

**send.go specifics:**
- `smtpTo`, `smtpCC`, `smtpBCC` are `[]string` (not `string`) bound with `StringSliceVar`, supporting comma-separated values or repeated flags.
- Recipients are also accepted as positional `args []string` in `sendMail`; they are appended to `viper.GetStringSlice("smtp.to")`.
- Body falls back to stdin via `readStdinBody` when `smtp.body` is empty. `readStdinBody` checks `cmd.InOrStdin() == os.Stdin` before calling `os.Stdin.Stat()`, so injecting a reader via `cmd.SetIn()` in tests bypasses the terminal check.

**imap.go specifics:**
- `imap list` defaults to listing messages (unseen only) using IMAP `FETCH ENVELOPE + BODY.PEEK[HEADER.FIELDS (Content-Type)]` — no full body download. Use `--all` for all messages, `--folders` for mailbox listing.
- The security status column (`S `/` E`/`SE`/`  `) is derived by `contentTypeSecurityStatus` from the Content-Type header. Detects S/MIME (`multipart/signed`, `application/pkcs7-mime`) and PGP/MIME (`multipart/signed`/`multipart/encrypted`). Inline RSA/ECDSA/GPG signatures require a body download to detect and show as plain.
- `imap list` uses `msg.SeqNum` (IMAP sequence number) for the `#ID` column. `SearchMessages` and `GetUnseenMessageIDs` return sequence numbers; `ReadMessages` fetches by sequence number. `ParseMessage` sets `MailType.ID = imapData.UID` (a different number) — do not conflate the two.
- `imap read --ids` and `imap delete --ids` both accept comma-separated sequence numbers. IDs shown by `imap list` can be used directly.
- `parseMessageIDs` is a shared helper used by both `imapReadMessages` and `imapDeleteMessages`.
- **IMAP Fetch deadlock pattern:** `client.Fetch` must always be run in a goroutine while the calling goroutine drains the message channel. A fixed-size buffered channel deadlocks whenever the mailbox has more messages than the buffer. Use an unbuffered channel and the pattern in `fetchListEntries`: `go func() { fetchErr <- c.Fetch(...) }()` then `for msg := range msgChan { ... }` then `<-fetchErr`. maillib v1.6.1 applies the same fix to `ReadMessages`; earlier versions deadlock on > 10 messages.
- `--query` flag (`imapSearchQuery` variable) uses IMAP `TEXT` criterion, which searches all headers and body — not body-only.

**config.go specifics:**
- `effectiveSettings()` calls `viper.AllSettings()` and strips `config` and `unit-test` keys (tool-internal, not meaningful in a config file).
- `nonZeroSettings()` recursively removes zero/false/empty/nil values via `reflect.Value.IsZero()` plus an explicit empty-slice check; used by `config save` to produce a minimal config file.
- `config save` writes with `0o600` permissions (may contain passwords).

**Signing integration:** `send.go:applySignature` reads `smtp.sign.*` viper keys. For S/MIME it sets `MailType.SignatureConfig` (handled by the library). For RSA/ECDSA/GPG it signs the body string and appends an inline block: `"\n\n--- <method> Signature ---\n<base64sig>"`. `imap.go:extractSignatureBlock` parses that same format during `imap read --verify-signature`.

**Mail library:** `github.com/tommi2day/gomodules/maillib` is an internal library that provides `SendMailConfigType`, `ImapType`, `MailType`, `SignMailContent`, `VerifyMailSignature`, and related types. `pwlib` provides RSA/ECDSA key generation used in tests.
- `ImapType.Client` is exported; `imap list` accesses it directly for the ENVELOPE fetch since the library has no header-only fetch method.
- `SearchMessages` / `GetUnseenMessageIDs` return sequence numbers (not UIDs). `ImapMsg.UID` / `ParseMessage` → `MailType.ID` is the IMAP UID — a different identifier.

## Testing

All tests are in `package cmd` (not `cmd_test`), so they access unexported variables directly to reset state between subtests. Each test file has a `reset*State()` function that zeros cobra flag variables and clears `pflag.Flag.Changed` — this is required because viper only falls back to config/env when a flag's `Changed` is false.

`test/testinit.go:InitTestDirs()` changes the working directory to `test/` so that viper's auto-discovery finds `test/mailcli.yaml`. Every test that exercises config loading must call `test.InitTestDirs()` first.

`--unit-test` flag (`unitTestFlag`) redirects logrus output to `RootCmd.OutOrStdout()` so test output is captured by `common.CmdRun`.

Docker integration tests (`TestMailDocker` in `mail_test.go`) spin up `docker.io/mailserver/docker-mailserver:15.1.0` via dockertest. Set `SKIP_MAIL=1` to skip them. Set `MAIL_HOST` to override the detected Docker host.

Shared test argument constants live in `cmd/testconst_test.go`. Command name constants (`cmdSend`, `cmdImap`, `cmdConfig`, etc.) and flag name constants (`flagMethod`, `flagBody`, etc.) are defined in the respective production source files and are accessible to test files in the same package.

**Reset functions per test file:**
- `config_test.go` — `resetRootState()`, `resetConfigCmdState()`
- `send_test.go` — `resetSMTPState()`
- `imap_test.go` — `resetImapState()` (resets persistent flags on `imapCmd`, plus per-subcommand flag sets including `imapListCmd`)

**stdin testing:** inject a reader via `sendCmd.SetIn(strings.NewReader(...))` before calling `common.CmdRun`. Defer `sendCmd.SetIn(nil)` to reset. `readStdinBody` detects the injected reader because `cmd.InOrStdin() != os.Stdin` and reads unconditionally.

## Linter

`golangci-lint` v2 config is in `.golangci.yml`. Key rules active beyond defaults: `goconst` (strings with ≥2 occurrences must be constants), `revive` (including `var-naming` — acronyms like SMTP must be all-caps), `lll` (max 200 chars), `gocyclo`/`gocognit` (max complexity 15), `goimports` formatter.
