package cmd

import (
	"os"
	"path/filepath"
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
			cmdSend,
			argSMTPTo,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
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
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPTo,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
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
			cmdImap, cmdList,
			argConfig, test.TestDir + "/no_config.yaml",
			argUnitTest,
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
			cmdSend,
			argConfig, test.TestDir + "/mailcli.yaml",
			argSMTPServerFlagHost,
			argSMTPTo,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
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
			cmdSend,
			argConfig, test.TestDir + "/no_config.yaml",
			argSMTPServerFlagHost,
			argSMTPTo,
			argSMTPSubjectTest,
			argSMTPBodyHello,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "send should fail")
		assert.Containsf(t, err.Error(), "flaghost.example.com", "flag should win over env var: %s", err)
		assert.NotContainsf(t, err.Error(), "envhost.example.com", "env var host must not be used: %s", err)
	})
}

func resetConfigCmdState() {
	configOutput = ""
	configSaveCmd.Flags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
	configCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		f.Changed = false
	})
}

func TestConfigCommand(t *testing.T) {
	test.InitTestDirs()

	t.Run("Config show prints file path and smtp/imap sections", func(t *testing.T) {
		resetRootState()
		resetConfigCmdState()
		args := []string{
			cmdConfig, cmdConfigShow,
			argConfig, test.TestDir + "/mailcli.yaml",
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, args)
		require.NoError(t, err)
		assert.Contains(t, out, "Config file:")
		assert.Contains(t, out, "mailcli.yaml")
		assert.Contains(t, out, "smtp:")
		assert.Contains(t, out, "imap:")
		assert.Contains(t, out, "127.0.0.1")
		assert.NotContains(t, out, "unit-test")
		assert.NotContains(t, out, "config:")
	})

	t.Run("Config show with non-existent config shows attempted path", func(t *testing.T) {
		resetRootState()
		resetConfigCmdState()
		args := []string{
			cmdConfig, cmdConfigShow,
			argConfig, test.TestDir + "/no_config.yaml",
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, args)
		require.NoError(t, err)
		// viper.ConfigFileUsed() returns the attempted path even when the read fails
		assert.Contains(t, out, "no_config.yaml")
	})

	t.Run("Config save writes non-zero settings to file", func(t *testing.T) {
		resetRootState()
		resetConfigCmdState()
		outFile := filepath.Join(t.TempDir(), "saved.yaml")
		args := []string{
			cmdConfig, cmdConfigSave,
			argConfig, test.TestDir + "/mailcli.yaml",
			"--output=" + outFile,
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, args)
		require.NoError(t, err)
		assert.Contains(t, out, "Config saved to")
		assert.Contains(t, out, outFile)

		data, readErr := os.ReadFile(outFile)
		require.NoError(t, readErr)
		content := string(data)
		assert.Contains(t, content, "smtp:")
		assert.Contains(t, content, "imap:")
		assert.Contains(t, content, "127.0.0.1")
		assert.NotContains(t, content, "unit-test")
		assert.NotContains(t, content, "config:")
		// zero-value defaults must be stripped
		assert.NotContains(t, content, "smtp.body")
		assert.NotContains(t, content, "smtp.subject")
	})

	t.Run("Config save default output filename", func(t *testing.T) {
		resetRootState()
		resetConfigCmdState()
		// Run in a temp dir so we don't pollute the working tree.
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		args := []string{
			cmdConfig, cmdConfigSave,
			argConfig, test.TestDir + "/mailcli.yaml",
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, args)
		require.NoError(t, err)
		assert.Contains(t, out, configName+"."+configType)
		_, statErr := os.Stat(filepath.Join(tmpDir, configName+"."+configType))
		assert.NoError(t, statErr, "default output file should exist")
	})
}
