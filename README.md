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

1. YAML config file — searched in `~/etc/mailcli.yaml` and `./mailcli.yaml`, or set explicitly with `--config`
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
mailcli imap read [--text=<filter>] [--save-attachments] ...
```

Displays unseen messages by default. Use `--text` to filter by subject, sender, or body content. Use `--save-attachments` to write attachments to `--imap.download-dir`.

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
