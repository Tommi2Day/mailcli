# Changelog mailcli

## [v1.0.0 - 2026-05-30]
### New
- initial release as Go CLI tool
- `send` command: send mail via SMTP with plain, STARTTLS, and SSL/TLS support
- `send` command: authentication methods plain, login, crammd5, xoauth2
- `send` command: attachments, CC/BCC, HTML body, custom HELO, max attachment size
- `imap list` command: list available mailboxes
- `imap status` command: show total/unseen message count and flags for a mailbox
- `imap read` command: read unseen messages, optional text filter, save attachments
- `imap search` command: search messages by subject/sender/body text
- `imap delete` command: permanently expunge messages by sequence IDs
- `version` command: print version, commit, and build date
- config file support via viper (`~/etc/mailcli.yaml`, `./mailcli.yaml`, or `--config`)
- environment variable support with prefix `MAILCLI_` (e.g. `MAILCLI_SMTP_SERVER`)
- `--debug` / `--info` / `--no-color` global flags
- goreleaser builds for linux, windows, darwin (amd64/arm64)
- deb/rpm packages via nfpm
- docker-mailserver integration tests
