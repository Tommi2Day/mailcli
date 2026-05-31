package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/mailcli/test"
)

// resetSMTPState resets cobra flag variables and pflag Changed state to avoid carryover between tests.
// Resetting Changed=false allows viper to fall back to yaml config and env when flags are not explicitly set.
func resetSMTPState() {
	smtpServer = ""
	smtpPort = 25
	smtpUsername = ""
	smtpPassword = ""
	smtpFrom = ""
	smtpTo = nil
	smtpCC = nil
	smtpBCC = nil
	smtpSubject = ""
	smtpBody = ""
	smtpAttach = ""
	smtpSSL = false
	smtpTLS = false
	smtpInsecure = false
	smtpAuthMethod = ""
	smtpContentHTML = false
	smtpTimeout = 0
	smtpHELO = ""
	smtpMaxSize = 0
	smtpSignMethod = ""
	smtpSignPrivKey = ""
	smtpSignPubKey = ""
	smtpSignPassphrase = ""
	smtpSignCertChain = ""
	smtpSignIncChain = false
	sendCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
}

func TestSendCommand(t *testing.T) {
	test.InitTestDirs()

	t.Run("Send missing server", func(t *testing.T) {
		resetSMTPState()
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPTo,
			argSMTPSubjectTest,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send without server should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Send missing recipient", func(t *testing.T) {
		resetSMTPState()
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPServer127,
			argSMTPSubjectTest,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send without recipient should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Send missing subject", func(t *testing.T) {
		resetSMTPState()
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPServer127,
			argSMTPTo,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send without subject should return an error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})
	t.Run("Send connection error to invalid port", func(t *testing.T) {
		resetSMTPState()
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPServer127,
			argSMTPPortInvalid,
			argSMTPTo,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send to invalid port should return an error")
		assert.Containsf(t, err.Error(), "send failed", "error should indicate send failed: %s", err)
		t.Logf("expected error: %v", err)
	})
	t.Run("Send body from stdin when flag is absent", func(t *testing.T) {
		resetSMTPState()
		sendCmd.SetIn(strings.NewReader("hello from stdin"))
		defer sendCmd.SetIn(nil)
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPServer127,
			argSMTPPortInvalid,
			argSMTPTo,
			argSMTPSubjectTest,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send to invalid port should return a connection error")
		assert.Containsf(t, err.Error(), "send failed", "should be a send error (body was read from stdin): %s", err)
	})
	t.Run("Send recipients as positional args", func(t *testing.T) {
		resetSMTPState()
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPServer127,
			argSMTPPortInvalid,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
			"--",
			"a@example.com", "b@example.com",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send to invalid port should return a connection error")
		assert.Containsf(t, err.Error(), "send failed", "should be a send error (recipients from positional args): %s", err)
	})
	t.Run("Send with config file sets server", func(t *testing.T) {
		resetSMTPState()
		// config provides smtp.server=127.0.0.1:31025 (not running)
		args := []string{
			cmdSend,
			argConfig, test.TestDir + "/mailcli.yaml",
			argSMTPTo,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send with config but no running server should error")
		assert.Containsf(t, err.Error(), "send failed", "should be a send error: %s", err)
		t.Logf("expected error: %v", err)
	})
}
