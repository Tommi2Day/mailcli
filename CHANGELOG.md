# Changelog mailcli

# [1.2.0 - 2026-06-22]
### Added
- apk packaging
### Changed
- send command is now default when no subcommand is given (e.g. `mailcli -s Subject addr1`); `send` subcommand still works as before; this allows `mailcli -s Subject addr1` without the extra `send` word, similar to mailx and other mail clients
### Fixed
- update SMTP auth method selection to prefer `AUTH=PLAIN` when the server supports it, falling back to `AUTH=LOGIN` if PLAIN is not offered; previously LOGIN was always used if the server supported it, even if PLAIN was also supported

## [1.1.1 - 2026-06-02]
### Added
- `config show` and `config save` now accept the same `--smtp.*` and `--imap.*` flags as the `send` and `imap` commands, with CLI values applied at highest priority (above config file and env vars). Example: `mailcli config save --smtp.server=relay.example.com --smtp.tls`
### Fixed
- IMAP message parsing: handle nil body error and enhance test coverage by upgrading to maillib 1.25.2

## [1.1.0 - 2026-06-01]
### Added
- **send: stdin body** — body text can be piped via stdin when `--smtp.body` is not set; flag/config/env value takes precedence over stdin
- **send: multiple recipients** — `--smtp.to`, `--smtp.cc`, and `--smtp.bcc` now accept a comma-separated list *or* the flag repeated multiple times (`--smtp.to=a@x.com --smtp.to=b@x.com`)
- **send: positional recipient arguments** — recipients can be given as positional arguments after all flags (mailx-style: `mailcli send -s Subject addr1 addr2`); mixed with `--smtp.to` is supported
- **imap list: message listing** — `imap list` now lists messages in the configured mailbox (unseen by default) instead of mailboxes; output shows sequence ID, date, security status, sender, and subject
- **imap list: `--all`** — show all messages in the mailbox, not just unseen ones
- **imap list: `--folders`** — restore the previous behaviour of listing mailboxes/folders
- **imap list: security indicator** — each message line shows a 2-character security field (`S ` = signed, ` E` = encrypted, `SE` = both) derived from the Content-Type header without downloading the full body; detects S/MIME and PGP/MIME
- **imap read: `--ids`** — read specific messages by comma-separated sequence IDs (e.g. `--ids=3,7`); takes precedence over `--text` and the default unseen filter; IDs match the `#` column in `imap list`
- **config show** — new subcommand that prints the config file in use and all resolved settings (flags + env + file merged) as valid YAML with inline comments describing each key; the config file path appears as a `#` comment so the entire output is parseable YAML
- **config save** — new subcommand that writes the resolved non-zero settings to a YAML config file with the same inline comments; defaults to `./mailcli.yaml`; path configurable via `--output`; written `0600`

### Changed

- `imap list` default behaviour changed from listing mailboxes to listing messages; use `imap list --folders` to list mailboxes
- **send: `-s`/`-S` swapped** — `-s` now sets the subject (mailx-compatible), `-S` sets the server; long flags `--smtp.subject` and `--smtp.server` are unchanged
- **send/imap: `-p`/`-P` swapped** — `-p` now sets the password, `-P` sets the port, on both `send` and `imap`; long flags unchanged
- **imap: `-S` for server** — `--imap.server` shorthand changed from `-s` to `-S` to match `send`; short flags are now consistent across both commands: `-S` server, `-P` port, `-p` password, `-u` username, `-T` timeout
- **send: `-b` for BCC, `-c` for CC** — `--smtp.bcc` gains `-b`, `--smtp.cc` gains `-c`; `--smtp.body` loses its `-b` shorthand (use `--smtp.body` or pipe via stdin)
- **global `--config` moved to `-F`** — frees `-c` for CC; `-F` signals "config file"
- **send/imap: `-T` for connect timeout** — `--smtp.timeout` and `--imap.timeout` gain the `-T` shorthand

### Fixed

- **imap search/read: `--text` renamed to `--query`** — flag now clearly signals a search string; description clarified to "any header or body (IMAP TEXT)" — the IMAP `TEXT` criterion was already searching all headers and body, but the previous description said "message body" only
- **imap search: `-T` shorthand conflict** — `imap search --text` had `-T` registered as a shorthand; `-T` is now the persistent `--imap.timeout` shorthand on the parent command, causing a cobra panic at runtime. Removed the `-T` shorthand from `--text`; use `--text=<query>` (long form).
- **imap list: deadlock on mailboxes with more than 10 messages** — `imap list` used a fixed-size channel of 10 for the IMAP FETCH response; go-imap's reader goroutine blocked sending the 11th message while `Fetch` waited for the command to complete. Fixed by running `Fetch` in a goroutine and draining the channel concurrently (unbuffered channel, goroutine pattern).
- **imap read: same deadlock in maillib** — upgraded `maillib` to v1.6.1 which applies the identical fix to `ReadMessages`; `imap read` on a mailbox with more than 10 unread messages no longer deadlocks.

## [1.0.0 - 2026-05-31]

Initial release.
