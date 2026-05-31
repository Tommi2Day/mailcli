# mailcli

Command-line tool for sending mail via SMTP and reading mail via IMAP.

[![codecov](https://codecov.io/gh/Tommi2Day/mailcli/graph/badge.svg?token=XYOGKC8RVO)](https://codecov.io/gh/Tommi2Day/mailcli)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/mailcli)

## Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [send — Send an email via SMTP](#send--send-an-email-via-smtp)
- [imap — Read and manage mail via IMAP](#imap--read-and-manage-mail-via-imap)
  - [imap list](#imap-list--list-messages-in-a-mailbox)
  - [imap status](#imap-status--show-mailbox-message-counts)
  - [imap read](#imap-read--read-full-messages)
  - [imap search](#imap-search--search-messages)
  - [imap delete](#imap-delete--permanently-delete-messages)
- [config — Manage configuration](#config--manage-configuration)
  - [config show](#config-show--print-the-active-configuration)
  - [config save](#config-save--save-the-effective-configuration)
- [sign — Sign content](#sign--sign-content-and-print-the-signature)
- [verify — Verify a signature](#verify--verify-a-content-signature)
- [Signing outgoing mail](#signing-outgoing-mail-send-command)
- [Global flags](#global-flags)
- [version](#version--print-version-information)

---

## Installation

Download the latest binary from the [releases page](https://github.com/tommi2day/mailcli/releases), or build from source:

```sh
go install github.com/tommi2day/mailcli@latest
```

---

## Configuration

Settings are resolved in this order (later sources override earlier ones):

1. YAML config file — searched in order, first found wins:
   - `./mailcli.yaml` (current directory)
   - `$HOME/.config/mailcli.yaml`
   - `$HOME/etc/mailcli.yaml`
   - `/etc/mailcli.yaml`
   - or set explicitly with `--config` / `-F`
2. Environment variables — prefix `MAILCLI_`, dots replaced by underscores (e.g. `MAILCLI_SMTP_SERVER`)
3. CLI flags

Example config file:

```yaml
smtp:
  server: mail.example.com
  port: 587
  username: user@example.com
  password: secret
  from: user@example.com
  tls: true

imap:
  server: mail.example.com
  port: 993
  username: user@example.com
  password: secret
  ssl: true
  inbox: INBOX
```

Use [`config save`](#config-save--save-the-effective-configuration) to generate a config file from the current effective settings.

---

## send — Send an email via SMTP

```sh
mailcli send [flags] [recipient...]
```

Recipients can be given via `--smtp.to`, as a comma-separated list, as the flag repeated, or as positional arguments after all flags (mailx-style). All methods can be combined.

The body can be provided via `--smtp.body`, read from config/env, or piped through stdin when the flag is not set.

| Flag | Description |
|------|-------------|
| `--smtp.server` / `-S` | SMTP server hostname or IP |
| `--smtp.port` / `-P` | SMTP server port (default 25) |
| `--smtp.username` / `-u` | Authentication username |
| `--smtp.password` / `-p` | Authentication password |
| `--smtp.from` / `-f` | Sender address |
| `--smtp.to` / `-t` | Recipient(s) — comma-separated or repeated; also positional args **(required)** |
| `--smtp.cc` / `-c` | CC recipient(s) — comma-separated or repeated |
| `--smtp.bcc` / `-b` | BCC recipient(s) — comma-separated or repeated |
| `--smtp.subject` / `-s` | Subject line **(required)** |
| `--smtp.body` | Body text (omit to read from stdin) |
| `--smtp.attach` / `-a` | Attachment file(s), comma-separated paths |
| `--smtp.ssl` | Use SMTPS (implicit TLS, typically port 465) |
| `--smtp.tls` | Use STARTTLS (typically port 587) |
| `--smtp.insecure` | Skip TLS/SSL certificate verification |
| `--smtp.auth` | Auth method: `plain` (default), `login`, `crammd5`, `xoauth2` |
| `--smtp.html` | Send body as `text/html` instead of `text/plain` |
| `--smtp.timeout` / `-T` | Connection timeout in seconds (default 15) |
| `--smtp.helo` | Custom HELO hostname |
| `--smtp.max-size` | Max attachment size in bytes (default 5 MB, 0 = unlimited) |

**Examples:**

```sh
# Plain-text mail (server and credentials from config file)
mailcli send -t alice@example.com -s "Hello" --smtp.body="Hi there"

# Positional recipients — mailx-style
mailcli send -s "Hello" alice@example.com bob@example.com

# Multiple recipients: flag + positional combined
mailcli send -s "Team update" -t alice@example.com bob@example.com carol@example.com

# CC and BCC
mailcli send -s "Meeting" -t alice@example.com -c bob@example.com -b carol@example.com \
  --smtp.body="See you there"

# Body from stdin
echo "nightly report" | mailcli send -s "Report" ops@example.com

git log --oneline -10 | mailcli send -s "Recent commits" dev@example.com

# Full explicit flags — STARTTLS with authentication
mailcli send \
  -S smtp.example.com -P 587 --smtp.tls -T 30 \
  -u me@example.com -p secret -f me@example.com \
  -t alice@example.com -s "Report" \
  --smtp.body="See attached" -a /tmp/report.pdf

# HTML mail via SMTPS (port 465)
mailcli send \
  -S mail.example.com -P 465 --smtp.ssl --smtp.insecure \
  -t team@example.com -s "Alert" \
  --smtp.body="<b>Disk full</b>" --smtp.html
```

---

## imap — Read and manage mail via IMAP

All `imap` subcommands share these connection flags:

| Flag | Description |
|------|-------------|
| `--imap.server` / `-S` | IMAP server hostname or IP |
| `--imap.port` / `-P` | IMAP server port (default 143) |
| `--imap.username` / `-u` | Username |
| `--imap.password` / `-p` | Password |
| `--imap.inbox` / `-i` | Mailbox to use (default `INBOX`) |
| `--imap.ssl` | Use IMAPS (implicit TLS, typically port 993) |
| `--imap.tls` | Use STARTTLS |
| `--imap.insecure` | Skip TLS/SSL certificate verification |
| `--imap.timeout` / `-T` | Connection timeout in seconds |
| `--imap.download-dir` / `-d` | Directory to save attachments (default `.`) |

### imap list — List messages in a mailbox

```sh
mailcli imap list [--all] [--folders] ...
```

By default shows unseen messages in the configured mailbox. Each line contains:

```
  #ID   Date              S/E  From                               Subject
  #3    2026-05-31 09:12      alice@example.com                  Re: Meeting
  #7    2026-05-31 14:05  S   Bob Smith <bob@corp.example>       Signed report
  #12   2026-05-31 17:00   E  noreply@service.com                Encrypted notice
```

The `S/E` column shows the message security status detected from the Content-Type header (no body download required):

| Value | Meaning |
|-------|---------|
| `S ` | Signed (S/MIME or PGP/MIME) |
| ` E` | Encrypted (S/MIME or PGP/MIME) |
| `SE` | Signed and encrypted |
| `  ` | Plain (or inline RSA/ECDSA/GPG — requires body to detect) |

The `#ID` sequence numbers can be passed directly to `imap read --ids` and `imap delete --ids`.

| Flag | Description |
|------|-------------|
| `--all` | List all messages, not just unseen |
| `--folders` | List mailboxes/folders instead of messages |

**Examples:**

```sh
# List unseen messages (connection from config file)
mailcli imap list

# List all messages in a specific folder
mailcli imap list --all -i Sent

# List available mailboxes
mailcli imap list --folders

# Explicit connection flags
mailcli imap list -S mail.example.com -P 993 --imap.ssl -u me@example.com -p secret
```

### imap status — Show mailbox message counts

```sh
mailcli imap status [-i <mailbox>] ...
```

Prints total message count, unseen count, and flags for the mailbox.

**Examples:**

```sh
# Status of INBOX (connection from config)
mailcli imap status

# Status of a specific folder
mailcli imap status -i Sent

# Explicit connection
mailcli imap status -S mail.example.com -P 993 --imap.ssl -u me@example.com -p secret -i INBOX
```

### imap read — Read full messages

```sh
mailcli imap read [--ids=<id,...>] [--query=<string>] [--save-attachments] [--verify-signature] ...
```

Priority for selecting messages: `--ids` → `--query` (search) → unseen (default).

| Flag | Description |
|------|-------------|
| `--ids` | Comma-separated sequence IDs to read (from `imap list` output) |
| `--query` | Show only messages matching this string in any header or body (IMAP TEXT) |
| `--save-attachments` | Write attachments to `--imap.download-dir` |
| `--verify-signature` | Verify inline RSA/ECDSA/GPG signatures or detect S/MIME |
| `--verify-public-key` | Public key or certificate for signature verification |
| `--verify-private-key` | Private key or S/MIME bundle (fallback) |
| `--verify-passphrase` | Passphrase for the verify key |

**Examples:**

```sh
# Read specific messages by ID (IDs come from imap list)
mailcli imap read --ids=3,7,12

# Read all unseen messages
mailcli imap read

# Filter by search string
mailcli imap read --query="invoice"

# Save attachments to a directory
mailcli imap read --save-attachments -d /tmp/mail

# Read and verify RSA/ECDSA/GPG inline signatures
mailcli imap read --verify-signature --verify-public-key=./mail.pub

# Combine: read specific message and verify its signature
mailcli imap read --ids=7 --verify-signature --verify-public-key=./mail.pub
```

### imap search — Search messages

```sh
mailcli imap search [--query=<string>] ...
```

Searches any header or body (IMAP TEXT criterion). Without `--query`, returns all unseen message IDs. Prints matching sequence IDs which can be piped to `imap read` or `imap delete`.

| Flag | Description |
|------|-------------|
| `--query` | Search string matched against any header or body (IMAP TEXT) |

**Examples:**

```sh
# Search by subject, sender, or body text
mailcli imap search --query="invoice"
mailcli imap search --query="alice@example.com"
mailcli imap search --query="urgent"

# List all unseen message IDs (no query)
mailcli imap search

# Search in a specific folder
mailcli imap search --query="report" -i Sent
```

### imap delete — Permanently delete messages

```sh
mailcli imap delete --ids=<id,...> ...
```

Permanently expunges the given sequence IDs from the mailbox. IDs match the `#ID` column in `imap list`.

| Flag | Description |
|------|-------------|
| `--ids` | Comma-separated sequence IDs to delete **(required)** |

**Examples:**

```sh
# Delete specific messages
mailcli imap delete --ids=3,7

# Find and delete in one step
mailcli imap search --query="spam" | xargs -I{} mailcli imap delete --ids={}
```

---

## config — Manage configuration

### config show — Print the active configuration

```sh
mailcli config show [-F <file>] [flags]
```

Prints the config file that was loaded and all resolved settings (file + environment variables + CLI flags merged) as valid YAML with inline comments. Useful for diagnosing which values are actually in effect.

**Examples:**

```sh
# Show effective config (auto-discovered config file)
mailcli config show

# Show config loaded from a specific file
mailcli config show -F /etc/mailcli.yaml

# Preview what env vars contribute
MAILCLI_SMTP_SERVER=smtp.example.com mailcli config show
```

### config save — Save the effective configuration

```sh
mailcli config save [--output=<path>] [flags]
```

Writes the resolved non-zero settings to a YAML config file with inline comments. Zero-value defaults are omitted. Written with `0600` permissions since the file may contain passwords.

| Flag | Description |
|------|-------------|
| `--output` / `-o` | Output file path (default `./mailcli.yaml`) |

**Examples:**

```sh
# Bootstrap a config file from environment variables
MAILCLI_SMTP_SERVER=smtp.example.com \
MAILCLI_SMTP_PORT=587 \
MAILCLI_SMTP_TLS=true \
MAILCLI_SMTP_USERNAME=me@example.com \
MAILCLI_SMTP_PASSWORD=secret \
MAILCLI_SMTP_FROM=me@example.com \
MAILCLI_IMAP_SERVER=imap.example.com \
MAILCLI_IMAP_PORT=993 \
MAILCLI_IMAP_SSL=true \
MAILCLI_IMAP_USERNAME=me@example.com \
MAILCLI_IMAP_PASSWORD=secret \
  mailcli config save --output=~/.config/mailcli.yaml

# Copy and override one value from an existing config
MAILCLI_SMTP_PASSWORD=new-secret \
  mailcli config save -F /etc/mailcli.yaml --output=~/.config/mailcli.yaml

# Preview what would be saved (without writing)
mailcli config show
```

---

## sign — Sign content and print the signature

```sh
mailcli sign --method=<method> --private-key=<file> --body=<text> [flags]
```

Signs a string and prints the base64-encoded signature. Supported methods: RSA, ECDSA, GPG, S/MIME.

| Flag | Description |
|------|-------------|
| `--method` / `-m` | Signing method: `rsa`, `ecdsa`, `gpg`, `smime` **(required)** |
| `--private-key` | Private key or S/MIME certificate bundle PEM file **(required)** |
| `--public-key` | Public key or certificate file |
| `--body` / `-b` | Content to sign **(required)** |
| `--passphrase` | Key passphrase |
| `--cert-chain` | Comma-separated cert chain PEM files (S/MIME) |
| `--include-chain` | Include cert chain in S/MIME signature |

Prints `Method: <method>` and `Signature: <base64>` to stdout.

**Examples:**

```sh
# RSA sign
mailcli sign -m rsa --private-key=./mail.key --public-key=./mail.pub \
  -b "hello world" --passphrase=secret

# ECDSA sign
mailcli sign -m ecdsa --private-key=./ec.key -b "hello world"

# S/MIME sign (cert+key bundle)
mailcli sign -m smime --private-key=./bundle.pem -b "hello world"

# Capture signature for use with verify
SIG=$(mailcli sign -m rsa --private-key=./mail.key -b "my message" | grep Signature | cut -d' ' -f2)
```

---

## verify — Verify a content signature

```sh
mailcli verify --method=<method> --body=<text> --signature=<base64> [flags]
```

Verifies a signature produced by `sign` or appended by `send --smtp.sign.*`. Prints `Signature valid (<method>)` or `Signature invalid (<method>)`.

| Flag | Description |
|------|-------------|
| `--method` / `-m` | Signing method: `rsa`, `ecdsa`, `gpg`, `smime` **(required)** |
| `--public-key` | Public key or certificate file **(required for rsa/ecdsa/smime)** |
| `--private-key` | S/MIME bundle fallback when no `--public-key` |
| `--body` / `-b` | Content to verify **(required)** |
| `--signature` | Base64-encoded signature **(required)** |
| `--passphrase` | Key passphrase |

**Examples:**

```sh
# RSA verify
mailcli verify -m rsa --public-key=./mail.pub \
  -b "hello world" --signature="$SIG"

# ECDSA verify
mailcli verify -m ecdsa --public-key=./ec.pub \
  -b "hello world" --signature="$SIG"

# S/MIME verify (bundle contains both cert and key)
mailcli verify -m smime --private-key=./bundle.pem \
  -b "hello world" --signature="$SIG"
```

---

## Signing outgoing mail (send command)

Add `--smtp.sign.*` flags to `send` to sign outgoing mail automatically.

| Flag | Description |
|------|-------------|
| `--smtp.sign.method` | Signing method: `rsa`, `ecdsa`, `gpg`, `smime` |
| `--smtp.sign.private-key` | Private key or S/MIME certificate bundle |
| `--smtp.sign.public-key` | Public key or certificate file |
| `--smtp.sign.passphrase` | Key passphrase |
| `--smtp.sign.cert-chain` | Comma-separated cert chain PEM files (S/MIME) |
| `--smtp.sign.include-chain` | Include cert chain in S/MIME signature |

For **S/MIME**, the mail is sent as a proper `multipart/signed` MIME message (RFC 5751). For **RSA/ECDSA/GPG**, the base64 signature is appended inline to the body, which `imap read --verify-signature` can verify.

**Examples:**

```sh
# Send S/MIME signed mail
mailcli send \
  -S mail.example.com -P 587 --smtp.tls \
  -f me@example.com -t you@example.com -s "Signed mail" \
  --smtp.body="Please verify this message." \
  --smtp.sign.method=smime --smtp.sign.private-key=./bundle.pem

# Send RSA-signed mail and later verify it via IMAP
mailcli send -t you@example.com -s "RSA signed" \
  --smtp.body="Authenticated content." \
  --smtp.sign.method=rsa \
  --smtp.sign.private-key=./mail.key \
  --smtp.sign.public-key=./mail.pub

# Recipient reads and verifies
mailcli imap read --verify-signature --verify-public-key=./mail.pub
```

---

## Global flags

These flags apply to every command:

| Flag | Description |
|------|-------------|
| `--config` / `-F` | Path to config file (overrides auto-discovery) |
| `--debug` | Verbose debug output |
| `--info` | Reduced info output |
| `--no-color` | Disable coloured log output |

---

## version — Print version information

```sh
mailcli version
```

Prints the build version, commit hash, and build date injected at release time.
