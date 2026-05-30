package cmd

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/mailcli/test"
)

func resetRootState() {
	cfgFile = ""
	RootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
}

// TestConfigPrecedence verifies config-file auto-discovery, env-var injection,
// and the override chain: config < env < CLI flag.
//
// The auto-discovery subtest must stay FIRST: viper caches the config file path
// after any --config flag is processed, and there is no public API to clear it
// without calling viper.Reset() (which also destroys BindPFlags registrations).
// config_test.go is lexically first in the package so TestConfigPrecedence runs
// before other test functions, and subtests execute in declaration order.
func TestConfigPrecedence(t *testing.T) {
	test.InitTestDirs()

	t.Run("Auto-discovery finds mailcli.yaml in current directory", func(t *testing.T) {
		resetSMTPState()
		resetRootState()
		// CWD is test/ (set by InitTestDirs); test/mailcli.yaml has smtp.server=127.0.0.1:31025.
		// No --config flag → viper searches the path list and picks up the file.
		args := []string{
			"send",
			"--smtp.to=test@example.com",
			"--smtp.subject=Test",
			"--smtp.body=Hello",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send should fail (server not running)")
		// "send failed" means a server address was found in the config and a
		// connection was attempted; "server is required" would mean discovery failed.
		assert.Containsf(t, err.Error(), "send failed", "auto-discovered config should provide smtp.server: %s", err)
	})

	t.Run("Env var sets smtp server", func(t *testing.T) {
		resetSMTPState()
		resetRootState()
		t.Setenv("MAILCLI_SMTP_SERVER", "envhost.example.com")
		args := []string{
			"send",
			"--config", test.TestDir + "/no_config.yaml",
			"--smtp.to=test@example.com",
			"--smtp.subject=Test",
			"--smtp.body=Hello",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send should fail connecting to env-var host")
		assert.Containsf(t, err.Error(), "envhost.example.com", "env var should supply the server: %s", err)
	})

	t.Run("Env var sets imap server", func(t *testing.T) {
		resetImapState()
		resetRootState()
		t.Setenv("MAILCLI_IMAP_SERVER", "imapenv.example.com")
		args := []string{
			"imap", "list",
			"--config", test.TestDir + "/no_config.yaml",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "imap list should fail connecting to env-var host")
		assert.Containsf(t, err.Error(), "imapenv.example.com", "env var should supply imap.server: %s", err)
	})

	t.Run("CLI flag overrides config file smtp server", func(t *testing.T) {
		resetSMTPState()
		resetRootState()
		// mailcli.yaml has smtp.server=127.0.0.1; the explicit flag must win.
		args := []string{
			"send",
			"--config", test.TestDir + "/mailcli.yaml",
			"--smtp.server=flaghost.example.com",
			"--smtp.to=test@example.com",
			"--smtp.subject=Test",
			"--smtp.body=Hello",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send should fail connecting to flag host")
		assert.Containsf(t, err.Error(), "flaghost.example.com", "flag should win over config file: %s", err)
		assert.NotContainsf(t, err.Error(), "127.0.0.1", "config file value must not be used: %s", err)
	})

	t.Run("CLI flag overrides env var smtp server", func(t *testing.T) {
		resetSMTPState()
		resetRootState()
		t.Setenv("MAILCLI_SMTP_SERVER", "envhost.example.com")
		args := []string{
			"send",
			"--config", test.TestDir + "/no_config.yaml",
			"--smtp.server=flaghost.example.com",
			"--smtp.to=test@example.com",
			"--smtp.subject=Test",
			"--smtp.body=Hello",
			"--unit-test",
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send should fail")
		assert.Containsf(t, err.Error(), "flaghost.example.com", "flag should win over env var: %s", err)
		assert.NotContainsf(t, err.Error(), "envhost.example.com", "env var host must not be used: %s", err)
	})
}
