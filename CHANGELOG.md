# Changelog mailcli

## [v1.1.0 - 2026-05-30]
### New
- `sign` command: sign content with rsa, ecdsa, gpg, or smime; prints base64 signature
- `verify` command: verify a base64 signature against content
- `send` command: add `--smtp.sign.*` flags to sign outgoing mail
  - S/MIME produces proper `multipart/signed` MIME (RFC 5751)
  - RSA/ECDSA/GPG appends a signature block to the body
- `imap read` command: add `--verify-signature` flag to verify message signatures
  - RSA/ECDSA/GPG inline blocks are cryptographically verified via `VerifyMailSignature`
  - S/MIME presence detected via `smime.p7s` attachment (cryptographic verify not available post-parse)
### Changed
- upgrade `gomodules` to v1.25.0 (adds `maillib.MailSignatureConfig`, `SignMailContent`, `VerifyMailSignature`, `IsValidSigningMethod`)
- upgrade `go-git/v5` to v1.19.1 (CVE-2026-34165, CVE-2026-33762)

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
