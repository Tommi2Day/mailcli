package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/mailcli/test"
)

// resetImapState resets cobra flag variables and pflag Changed state to avoid carryover between tests.
// Resetting Changed=false allows viper to fall back to yaml config and env when flags are not explicitly set.
func resetImapState() {
	imapServer = ""
	imapPort = 143
	imapUsername = ""
	imapPassword = ""
	imapInbox = "INBOX"
	imapSSL = false
	imapTLS = false
	imapInsecure = false
	imapTimeout = 0
	imapDownloadDir = "."
	imapSearchText = ""
	imapMessageIDs = ""
	imapSaveAttach = false
	imapCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
	imapReadCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
	imapDeleteCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
	imapSearchCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
}

func TestImapCommand(t *testing.T) {
	test.InitTestDirs()

	t.Run("Imap list missing server", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "list",
			"--config", test.TestDir + "/no_config.yaml",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap list without server should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Imap status missing server", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "status",
			"--config", test.TestDir + "/no_config.yaml",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap status without server should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Imap read missing server", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "read",
			"--config", test.TestDir + "/no_config.yaml",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap read without server should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Imap delete missing ids", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "delete",
			"--config", test.TestDir + "/no_config.yaml",
			"--imap.server=127.0.0.1",
			"--imap.port=19999",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap delete without ids should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Imap delete missing server", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "delete",
			"--config", test.TestDir + "/no_config.yaml",
			"--ids=1",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap delete without server should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Imap connect error to invalid port", func(t *testing.T) {
		resetImapState()
		args := []string{
			"imap", "list",
			"--config", test.TestDir + "/no_config.yaml",
			"--imap.server=127.0.0.1",
			"--imap.port=19999",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap to invalid port should return an error")
		assert.Containsf(t, err.Error(), "connect", "error should mention connect: %s", err)
		t.Logf("expected error: %v", err)
	})
	t.Run("Imap with config file sets server", func(t *testing.T) {
		resetImapState()
		// config provides imap.server=127.0.0.1:31993 with SSL (not running)
		args := []string{
			"imap", "list",
			"--config", test.TestDir + "/mailcli.yaml",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap with config but no running server should error")
		assert.Containsf(t, err.Error(), "connect", "should be a connect error: %s", err)
		t.Logf("expected error: %v", err)
	})
}
