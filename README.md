# mailcli

Command-line tool for sending mail via SMTP and reading mail via IMAP.

![CI](https://github.com/tommi2day/mailcli/actions/workflows/main.yml/badge.svg)
[![codecov](https://codecov.io/gh/Tommi2Day/mailcli/branch/main/graph/badge.svg?token=3EBK75VLC8)](https://codecov.io/gh/Tommi2Day/mailcli)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tommi2day/mailcli)

## Installation

Download the latest binary from the [releases page](https://github.com/tommi2day/mailcli/releases), or build from source:

```sh
go install github.com/tommi2day/mailcli@latest
```

## Configuration

Settings are resolved in this order (later sources override earlier ones):

1. YAML config file — searched in order, first found wins:
   - `./mailcli.yaml` (current directory)
   - `$HOME/.config/mailcli.yaml`
   - `$HOME/etc/mailcli.yaml`
   - `/etc/mailcli.yaml`
   - or set explicitly with `--config`
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
  insecure: false

imap:
  server: mail.example.com
  port: 993
  username: user@example.com
  password: secret
  ssl: true
  insecure: false
  inbox: INBOX
```

## Commands

### send — Send an email via SMTP

```sh
mailcli send [flags]
```

| Flag | Description |
|------|-------------|
| `--smtp.server` / `-s` | SMTP server hostname or IP |
| `--smtp.port` / `-p` | SMTP server port (default 25) |
| `--smtp.username` / `-u` | Authentication username |
| `--smtp.password` / `-P` | Authentication password |
| `--smtp.from` / `-f` | Sender address |
| `--smtp.to` / `-t` | Recipient address(es), comma-separated **(required)** |
| `--smtp.cc` | CC recipient(s), comma-separated |
| `--smtp.bcc` | BCC recipient(s), comma-separated |
| `--smtp.subject` / `-S` | Subject line **(required)** |
| `--smtp.body` / `-b` | Body text |
| `--smtp.attach` / `-a` | Attachment file(s), comma-separated paths |
| `--smtp.ssl` | Use SMTPS (implicit TLS, typically port 465) |
| `--smtp.tls` | Use STARTTLS (typically port 587) |
| `--smtp.insecure` | Skip TLS/SSL certificate verification |
| `--smtp.auth` | Auth method: `plain` (default), `login`, `crammd5`, `xoauth2` |
| `--smtp.html` | Send body as `text/html` instead of `text/plain` |
| `--smtp.timeout` | Connection timeout in seconds (default 15) |
| `--smtp.helo` | Custom HELO hostname |
| `--smtp.max-size` | Max attachment size in bytes (default 5 MB, 0 = unlimited) |

**Examples:**

```sh
# Send a plain-text mail using server from config
mailcli send --smtp.to=alice@example.com --smtp.subject="Hello" --smtp.body="Hi there"

# Send with STARTTLS and authentication, explicit server
mailcli send \
  --smtp.server=smtp.example.com --smtp.port=587 \
  --smtp.username=me@example.com --smtp.password=secret \
  --smtp.tls --smtp.from=me@example.com \
  --smtp.to=alice@example.com --smtp.subject="Report" \
  --smtp.body="See attached" --smtp.attach=/tmp/report.pdf

# Send HTML mail via SMTPS (port 465)
mailcli send \
  --smtp.server=mail.example.com --smtp.port=465 \
  --smtp.ssl --smtp.insecure \
  --smtp.to=team@example.com --smtp.subject="Alert" \
  --smtp.body="<b>Disk full</b>" --smtp.html
```

---

### imap — Read and manage mail via IMAP

All `imap` subcommands share these connection flags:

| Flag | Description |
|------|-------------|
| `--imap.server` / `-s` | IMAP server hostname or IP |
| `--imap.port` / `-p` | IMAP server port (default 143) |
| `--imap.username` / `-u` | Username |
| `--imap.password` / `-P` | Password |
| `--imap.inbox` / `-i` | Mailbox to use (default `INBOX`) |
| `--imap.ssl` | Use IMAPS (implicit TLS, typically port 993) |
| `--imap.tls` | Use STARTTLS |
| `--imap.insecure` | Skip TLS/SSL certificate verification |
| `--imap.timeout` | Connection timeout in seconds |
| `--imap.download-dir` / `-d` | Directory to save attachments (default `.`) |

#### imap list — List mailboxes

```sh
mailcli imap list --imap.server=mail.example.com --imap.port=993 \
  --imap.username=me@example.com --imap.password=secret --imap.ssl
```

#### imap status — Show mailbox message counts

```sh
mailcli imap status --imap.server=mail.example.com ...
```

Output: total messages, unseen count, and flags for the mailbox.

#### imap read — Read messages

```sh
mailcli imap read [--text=<filter>] [--save-attachments] [--verify-signature] ...
```

Displays unseen messages by default. Use `--text` to filter by subject, sender, or body content. Use `--save-attachments` to write attachments to `--imap.download-dir`.

Add `--verify-signature` to verify any inline signature block (RSA/ECDSA/GPG) or detect S/MIME signatures in the received message.

| Flag | Description |
|------|-------------|
| `--verify-signature` | Verify message signatures if present |
| `--verify-public-key` | Public key or certificate for verification |
| `--verify-private-key` | Private key or S/MIME bundle (fallback) |
| `--verify-passphrase` | Passphrase for the verify key |

For **RSA/ECDSA/GPG** messages signed with `send --smtp.sign.*`, the inline signature block is cryptographically verified and the result is printed as `Signature-Status: valid` or `Signature-Status: invalid`. For **S/MIME**, the presence of `smime.p7s` is detected and reported — cryptographic verification requires the raw MIME body which is unavailable after IMAP parsing.

#### imap search — Search messages

```sh
mailcli imap search --text="invoice" ...
```

Searches message subject, sender, and body. Without `--text`, returns all unseen messages. Prints matching message IDs.

#### imap delete — Permanently delete messages

```sh
mailcli imap delete --ids=1,2,5 ...
```

Permanently expunges the given sequence IDs from the mailbox.

---

### sign — Sign content and print the signature

```sh
mailcli sign --method=<method> --private-key=<file> --body=<text> [flags]
```

| Flag | Description |
|------|-------------|
| `--method` / `-m` | Signing method: `rsa`, `ecdsa`, `gpg`, `smime` **(required)** |
| `--private-key` | Private key or S/MIME certificate bundle PEM file **(required)** |
| `--public-key` | Public key or certificate file (optional; used for self-verification output) |
| `--body` / `-b` | Content to sign **(required)** |
| `--passphrase` | Key passphrase |
| `--cert-chain` | Comma-separated cert chain PEM files (S/MIME) |
| `--include-chain` | Include cert chain in S/MIME signature |

Prints `Method:` and `Signature:` (base64-encoded) to stdout.

**Examples:**

```sh
# RSA sign
mailcli sign --method=rsa --private-key=./mail.key --public-key=./mail.pub \
  --body="hello world" --passphrase=secret

# S/MIME sign (cert+key bundle)
mailcli sign --method=smime --private-key=./bundle.pem --body="hello world"
```

---

### verify — Verify a content signature

```sh
mailcli verify --method=<method> --body=<text> --signature=<base64> [flags]
```

| Flag | Description |
|------|-------------|
| `--method` / `-m` | Signing method: `rsa`, `ecdsa`, `gpg`, `smime` **(required)** |
| `--public-key` | Public key or certificate file **(required for rsa/ecdsa/smime)** |
| `--private-key` | S/MIME bundle fallback when no `--public-key` |
| `--body` / `-b` | Content to verify **(required)** |
| `--signature` | Base64-encoded signature **(required)** |
| `--passphrase` | Key passphrase |

Prints `Signature valid (<method>)` or `Signature invalid (<method>)`.

**Example:**

```sh
mailcli verify --method=rsa --public-key=./mail.pub \
  --body="hello world" --signature="<base64sig>"
```

---

### Signing outgoing mail (send command)

Add `--smtp.sign.*` flags to the `send` command to sign outgoing mail.

| Flag | Description |
|------|-------------|
| `--smtp.sign.method` | Signing method: `rsa`, `ecdsa`, `gpg`, `smime` |
| `--smtp.sign.private-key` | Private key or S/MIME certificate bundle |
| `--smtp.sign.public-key` | Public key or certificate file (S/MIME) |
| `--smtp.sign.passphrase` | Key passphrase |
| `--smtp.sign.cert-chain` | Comma-separated cert chain PEM files (S/MIME) |
| `--smtp.sign.include-chain` | Include cert chain in S/MIME signature |

For **S/MIME**, the mail is sent as a proper `multipart/signed` MIME message (RFC 5751). For **RSA/ECDSA/GPG**, the base64 signature is appended to the body as a signature block.

**Example:**

```sh
# Send S/MIME signed mail
mailcli send \
  --smtp.server=mail.example.com --smtp.port=587 --smtp.tls \
  --smtp.from=me@example.com --smtp.to=you@example.com \
  --smtp.subject="Signed mail" --smtp.body="See attached signature" \
  --smtp.sign.method=smime --smtp.sign.private-key=./bundle.pem
```

---

### Global flags

| Flag | Description |
|------|-------------|
| `--config` / `-c` | Path to config file |
| `--debug` | Verbose debug output |
| `--info` | Reduced info output |
| `--no-color` | Disable colored log output |

---

### version — Print version information

```sh
mailcli version
```
