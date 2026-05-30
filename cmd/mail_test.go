package cmd

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/mailcli/test"
)

const mailFrom = "root@test.local"
const mailTo = "info@test.local"
const mailUser = "info@test.local"
const mailPass = "testpass"
const rootUser = "root@test.local"
const rootPass = "testpass"

func TestMailDocker(t *testing.T) {
	if os.Getenv("SKIP_MAIL") != "" {
		t.Skip("Skipping Mail Docker testing (SKIP_MAIL set)")
	}
	test.InitTestDirs()

	var err error
	mailContainer, err = prepareMailContainer()
	defer common.DestroyDockerContainer(mailContainer)
	require.NoErrorf(t, err, "Mailserver not available: %s", err)
	require.NotNil(t, mailContainer, "Prepare failed")

	smtpServer := mailDockerServer
	imapServer := mailDockerServer

	t.Run("Send Mail anonym SMTP 25", func(t *testing.T) {
		resetSMTPState()
		h := time.Now()
		args := []string{
			"send",
			fmt.Sprintf("--smtp.server=%s", smtpServer),
			fmt.Sprintf("--smtp.port=%d", smtpDockerPort),
			fmt.Sprintf("--smtp.from=%s", mailFrom),
			fmt.Sprintf("--smtp.to=%s", mailTo),
			"--smtp.subject=Docker Test 1",
			fmt.Sprintf("--smtp.body=Test at %s", h.Format("15:04:05")),
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "send anonym returned error: %v", err)
		assert.Containsf(t, out, "successfully", "output should confirm success")
		t.Log(out)
	})

	t.Run("Send Mail STARTTLS 25", func(t *testing.T) {
		resetSMTPState()
		h := time.Now()
		args := []string{
			"send",
			fmt.Sprintf("--smtp.server=%s", smtpServer),
			fmt.Sprintf("--smtp.port=%d", smtpDockerPort),
			fmt.Sprintf("--smtp.from=%s", mailFrom),
			fmt.Sprintf("--smtp.to=%s", mailTo),
			"--smtp.tls",
			"--smtp.insecure",
			"--smtp.subject=Docker Test 2",
			fmt.Sprintf("--smtp.body=TLS Test at %s", h.Format("15:04:05")),
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "send TLS returned error: %v", err)
		t.Log(out)
	})

	t.Run("Send Mail SSL 465 login auth with attachment", func(t *testing.T) {
		resetSMTPState()
		h := time.Now()
		attachFile := test.TestDir + "/docker/mail/ssl/ca.crt"
		args := []string{
			"send",
			fmt.Sprintf("--smtp.server=%s", smtpServer),
			fmt.Sprintf("--smtp.port=%d", sslDockerPort),
			fmt.Sprintf("--smtp.from=%s", rootUser),
			fmt.Sprintf("--smtp.to=%s", mailTo),
			fmt.Sprintf("--smtp.username=%s", rootUser),
			fmt.Sprintf("--smtp.password=%s", rootPass),
			"--smtp.ssl",
			"--smtp.insecure",
			"--smtp.auth=login",
			fmt.Sprintf("--smtp.attach=%s", attachFile),
			"--smtp.subject=Docker Test 3",
			fmt.Sprintf("--smtp.body=SSL+attach Test at %s", h.Format("15:04:05")),
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "send SSL returned error: %v", err)
		t.Log(out)
	})

	t.Run("Send Mail STARTTLS 587", func(t *testing.T) {
		resetSMTPState()
		h := time.Now()
		args := []string{
			"send",
			fmt.Sprintf("--smtp.server=%s", smtpServer),
			fmt.Sprintf("--smtp.port=%d", tlsDockerPort),
			fmt.Sprintf("--smtp.from=%s", rootUser),
			fmt.Sprintf("--smtp.to=%s", mailTo),
			fmt.Sprintf("--smtp.username=%s", rootUser),
			fmt.Sprintf("--smtp.password=%s", rootPass),
			"--smtp.tls",
			"--smtp.insecure",
			"--smtp.subject=Docker Test 4",
			fmt.Sprintf("--smtp.body=TLS 587 Test at %s", h.Format("15:04:05")),
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "send TLS 587 returned error: %v", err)
		t.Log(out)
	})

	t.Log("Wait for mails to be delivered...")
	time.Sleep(10 * time.Second)

	t.Run("Imap list mailboxes", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "list",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "imap list returned error: %v", err)
		assert.Containsf(t, out, "INBOX", "output should contain INBOX")
		t.Log(out)
	})

	t.Run("Imap status", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "status",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "imap status returned error: %v", err)
		assert.Containsf(t, out, "INBOX", "output should contain INBOX")
		assert.Containsf(t, out, "Total:", "output should contain Total:")
		t.Log(out)
	})

	t.Run("Imap search by subject text", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "search",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			"--text=Docker Test",
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "imap search returned error: %v", err)
		assert.Containsf(t, out, "Found", "search should find messages")
		t.Log(out)
	})

	t.Run("Imap read unseen messages", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "read",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "imap read returned error: %v", err)
		assert.Containsf(t, out, "Subject:", "output should contain Subject:")
		t.Log(out)
	})

	t.Run("Imap read with save attachments", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "read",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			fmt.Sprintf("--imap.download-dir=%s", test.TestData),
			"--save-attachments",
			"--text=Docker Test 3",
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, args)
		assert.NoErrorf(t, err, "imap read with attachments returned error: %v", err)
		assert.Containsf(t, out, "Attach:", "output should contain attachment info")
		t.Log(out)
	})

	t.Run("Imap delete messages", func(t *testing.T) {
		resetImapState()
		// search all messages (seen or unseen) by subject text
		searchArgs := []string{
			"imap", "search",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			"--text=Docker Test",
			"--unit-test",
		}
		out, err := common.CmdRun(RootCmd, searchArgs)
		assert.NoErrorf(t, err, "search for delete test returned error: %v", err)
		t.Logf("search result: %s", out)

		resetImapState()
		deleteArgs := []string{
			"imap", "delete",
			fmt.Sprintf("--imap.server=%s", imapServer),
			fmt.Sprintf("--imap.port=%d", imapsDockerPort),
			fmt.Sprintf("--imap.username=%s", mailUser),
			fmt.Sprintf("--imap.password=%s", mailPass),
			"--imap.ssl",
			"--imap.insecure",
			"--ids=1",
			"--unit-test",
		}
		out, err = common.CmdRun(RootCmd, deleteArgs)
		assert.NoErrorf(t, err, "imap delete returned error: %v", err)
		assert.Containsf(t, out, "Deleted", "output should confirm deletion")
		t.Log(out)
	})
}
