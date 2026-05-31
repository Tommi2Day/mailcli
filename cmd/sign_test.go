package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tommi2day/gomodules/common"
	"github.com/tommi2day/gomodules/pwlib"
	"github.com/tommi2day/mailcli/test"
)

const signaturePrefix = "Signature: "

func resetSignState() {
	signMethod = ""
	signPrivKey = ""
	signPubKey = ""
	signPassphrase = ""
	signCertChain = ""
	signIncChain = false
	signBody = ""
	signCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
}

func resetVerifyState() {
	verifyMethod = ""
	verifyPubKey = ""
	verifyPrivKey = ""
	verifyPassphrase = ""
	verifyBody = ""
	verifySignature = ""
	verifyCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
}

func TestSignCommand(t *testing.T) {
	test.InitTestDirs()

	t.Run("Sign missing method", func(t *testing.T) {
		resetSignState()
		args := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argBodyHelloWorld,
			argPrivKeyTmp,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "sign without method should error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})

	t.Run("Sign missing body", func(t *testing.T) {
		resetSignState()
		args := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			argPrivKeyTmp,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "sign without body should error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})

	t.Run("Sign missing private key", func(t *testing.T) {
		resetSignState()
		args := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			argBodyHelloWorld,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "sign without private key should error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})

	t.Run("Sign invalid method", func(t *testing.T) {
		resetSignState()
		args := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodBadMethod,
			argBodyHello,
			argPrivKeyTmp,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "sign with invalid method should error")
		assert.Containsf(t, err.Error(), "invalid signing method", "error should describe invalid method: %s", err)
	})

	t.Run("Sign with nonexistent key file", func(t *testing.T) {
		resetSignState()
		args := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			argBodyHelloWorld,
			"--private-key=/nonexistent/path/key.pem",
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "sign with missing key file should error")
		assert.Containsf(t, err.Error(), "signing failed", "error should mention signing failed: %s", err)
	})

	t.Run("Sign and verify RSA round-trip", func(t *testing.T) {
		pubKeyFile := test.TestData + "/sign_test_rsa.pub"
		privKeyFile := test.TestData + "/sign_test_rsa.key"
		keyPass := mailPass

		_, _, err := pwlib.GenRsaKey(pubKeyFile, privKeyFile, keyPass)
		require.NoErrorf(t, err, "GenRsaKey failed: %v", err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		const content = "test message for signing"

		// Sign
		resetSignState()
		signArgs := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			"--body=" + content,
			"--private-key=" + privKeyFile,
			"--public-key=" + pubKeyFile,
			"--passphrase=" + keyPass,
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, signArgs)
		require.NoErrorf(t, err, "sign returned error: %v", err)
		assert.Containsf(t, out, "Signature:", "output should contain signature")
		t.Logf("sign output: %s", out)

		// Extract signature from output
		var sig string
		for _, line := range strings.Split(out, "\n") {
			if len(line) > len(signaturePrefix) && line[:len(signaturePrefix)] == signaturePrefix {
				sig = line[len(signaturePrefix):]
				break
			}
		}
		require.NotEmpty(t, sig, "extracted signature must not be empty")

		// Verify
		resetVerifyState()
		verifyArgs := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			"--body=" + content,
			"--public-key=" + pubKeyFile,
			"--signature=" + sig,
			argUnitTest,
		}
		out, err = common.CmdRun(RootCmd, verifyArgs)
		require.NoErrorf(t, err, "verify returned error: %v", err)
		assert.Containsf(t, out, "valid", "output should confirm valid signature")
		t.Logf("verify output: %s", out)
	})

	t.Run("Sign and verify ECDSA round-trip", func(t *testing.T) {
		pubKeyFile := test.TestData + "/sign_test_ecdsa.pub"
		privKeyFile := test.TestData + "/sign_test_ecdsa.key"
		keyPass := "ecpass"

		_, _, err := pwlib.GenEcdsaKey(pubKeyFile, privKeyFile, keyPass)
		require.NoErrorf(t, err, "GenEcdsaKey failed: %v", err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		const content = "ecdsa test message"

		resetSignState()
		signArgs := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodECDSA,
			"--body=" + content,
			"--private-key=" + privKeyFile,
			"--public-key=" + pubKeyFile,
			"--passphrase=" + keyPass,
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, signArgs)
		require.NoErrorf(t, err, "ecdsa sign returned error: %v", err)

		var sig string
		for _, line := range strings.Split(out, "\n") {
			if len(line) > len(signaturePrefix) && line[:len(signaturePrefix)] == signaturePrefix {
				sig = line[len(signaturePrefix):]
				break
			}
		}
		require.NotEmpty(t, sig)

		resetVerifyState()
		verifyArgs := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodECDSA,
			"--body=" + content,
			"--public-key=" + pubKeyFile,
			"--signature=" + sig,
			argUnitTest,
		}
		out, err = common.CmdRun(RootCmd, verifyArgs)
		require.NoErrorf(t, err, "ecdsa verify returned error: %v", err)
		assert.Containsf(t, out, "valid", "ecdsa signature should be valid")
	})
}

func TestVerifyCommand(t *testing.T) {
	test.InitTestDirs()

	t.Run("Verify missing method", func(t *testing.T) {
		resetVerifyState()
		args := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argBodyHello,
			argSignatureABC,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "verify without method should error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})

	t.Run("Verify missing body", func(t *testing.T) {
		resetVerifyState()
		args := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			argSignatureABC,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "verify without body should error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})

	t.Run("Verify missing signature", func(t *testing.T) {
		resetVerifyState()
		args := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			argBodyHello,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "verify without signature should error")
		assert.Containsf(t, err.Error(), "required", "error should mention required: %s", err)
	})

	t.Run("Verify invalid method", func(t *testing.T) {
		resetVerifyState()
		args := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodBadMethod,
			argBodyHello,
			argSignatureABC,
			argUnitTest,
		}
		_, err := common.CmdRun(RootCmd, args)
		require.Errorf(t, err, "verify with invalid method should error")
		assert.Containsf(t, err.Error(), "invalid signing method", "error should describe method: %s", err)
	})

	t.Run("Verify tampered content fails", func(t *testing.T) {
		pubKeyFile := test.TestData + "/verify_test_rsa.pub"
		privKeyFile := test.TestData + "/verify_test_rsa.key"
		keyPass := mailPass

		_, _, err := pwlib.GenRsaKey(pubKeyFile, privKeyFile, keyPass)
		require.NoErrorf(t, err, "GenRsaKey failed: %v", err)
		defer os.Remove(pubKeyFile)
		defer os.Remove(privKeyFile)

		// Sign original content
		resetSignState()
		signArgs := []string{
			cmdSign,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			"--body=original content",
			"--private-key=" + privKeyFile,
			"--public-key=" + pubKeyFile,
			"--passphrase=" + keyPass,
			argUnitTest,
		}
		out, err := common.CmdRun(RootCmd, signArgs)
		require.NoErrorf(t, err, "sign failed: %v", err)

		var sig string
		for _, line := range strings.Split(out, "\n") {
			if len(line) > len(signaturePrefix) && line[:len(signaturePrefix)] == signaturePrefix {
				sig = line[len(signaturePrefix):]
				break
			}
		}
		require.NotEmpty(t, sig)

		// Verify with tampered body — must fail or report invalid
		resetVerifyState()
		verifyArgs := []string{
			cmdVerify,
			argConfig, test.TestDir + "/no_config.yaml",
			argMethodRSA,
			"--body=tampered content",
			"--public-key=" + pubKeyFile,
			"--signature=" + sig,
			argUnitTest,
		}
		out, err = common.CmdRun(RootCmd, verifyArgs)
		// Either error or "invalid" in output
		if err == nil {
			assert.Containsf(t, out, "invalid", "tampered content should not verify as valid")
		}
		t.Logf("tampered verify result: out=%s err=%v", out, err)
	})
}
