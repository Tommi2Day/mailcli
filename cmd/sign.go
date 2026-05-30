// Package cmd commands
package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tommi2day/gomodules/maillib"
)

var (
	signMethod     = ""
	signPrivKey    = ""
	signPubKey     = ""
	signPassphrase = ""
	signCertChain  = ""
	signIncChain   = false
	signBody       = ""

	verifyMethod     = ""
	verifyPubKey     = ""
	verifyPrivKey    = ""
	verifyPassphrase = ""
	verifyBody       = ""
	verifySignature  = ""

	signCmd = &cobra.Command{
		Use:          "sign",
		Short:        "Sign content and print the signature",
		RunE:         runSign,
		SilenceUsage: true,
	}

	verifyCmd = &cobra.Command{
		Use:          "verify",
		Short:        "Verify a mail content signature",
		RunE:         runVerify,
		SilenceUsage: true,
	}
)

func init() {
	signCmd.Flags().StringVarP(&signMethod, "method", "m", "", "signing method: rsa, ecdsa, gpg, smime (required)")
	signCmd.Flags().StringVar(&signPrivKey, "private-key", "", "private key or S/MIME certificate bundle file (required)")
	signCmd.Flags().StringVar(&signPubKey, "public-key", "", "public key or certificate file")
	signCmd.Flags().StringVarP(&signBody, "body", "b", "", "content to sign (required)")
	signCmd.Flags().StringVar(&signPassphrase, "passphrase", "", "key passphrase")
	signCmd.Flags().StringVar(&signCertChain, "cert-chain", "", "comma-separated cert chain PEM files (S/MIME)")
	signCmd.Flags().BoolVar(&signIncChain, "include-chain", false, "include cert chain in S/MIME signature")
	bindNamedFlags(signCmd, map[string]string{
		"sign.method":        "method",
		"sign.private-key":   "private-key",
		"sign.public-key":    "public-key",
		"sign.body":          "body",
		"sign.passphrase":    "passphrase",
		"sign.cert-chain":    "cert-chain",
		"sign.include-chain": "include-chain",
	})

	verifyCmd.Flags().StringVarP(&verifyMethod, "method", "m", "", "signing method: rsa, ecdsa, gpg, smime (required)")
	verifyCmd.Flags().StringVar(&verifyPubKey, "public-key", "", "public key or certificate file")
	verifyCmd.Flags().StringVar(&verifyPrivKey, "private-key", "", "certificate bundle (S/MIME fallback when no public-key)")
	verifyCmd.Flags().StringVarP(&verifyBody, "body", "b", "", "content to verify (required)")
	verifyCmd.Flags().StringVar(&verifyPassphrase, "passphrase", "", "key passphrase")
	verifyCmd.Flags().StringVar(&verifySignature, "signature", "", "base64-encoded signature to verify (required)")
	bindNamedFlags(verifyCmd, map[string]string{
		"verify.method":      "method",
		"verify.public-key":  "public-key",
		"verify.private-key": "private-key",
		"verify.body":        "body",
		"verify.passphrase":  "passphrase",
		"verify.signature":   "signature",
	})

	RootCmd.AddCommand(signCmd)
	RootCmd.AddCommand(verifyCmd)
}

// bindNamedFlags binds cobra flags to namespaced viper keys so sign.* and verify.*
// do not collide in viper's global key space.
func bindNamedFlags(cmd *cobra.Command, bindings map[string]string) {
	for viperKey, flagName := range bindings {
		if err := viper.BindPFlag(viperKey, cmd.Flags().Lookup(flagName)); err != nil {
			log.Fatal(err)
		}
	}
}

func runSign(cmd *cobra.Command, _ []string) error {
	method := viper.GetString("sign.method")
	body := viper.GetString("sign.body")
	privKey := viper.GetString("sign.private-key")

	if method == "" {
		return fmt.Errorf("signing method is required (--method)")
	}
	if body == "" {
		return fmt.Errorf("content is required (--body)")
	}
	if privKey == "" {
		return fmt.Errorf("private key is required (--private-key)")
	}
	if !maillib.IsValidSigningMethod(method) {
		return fmt.Errorf("invalid signing method %q: valid methods are %s",
			method, strings.Join(maillib.GetSupportedSigningMethods(), ", "))
	}

	cfg := buildSignConfig()
	sig, err := maillib.SignMailContent(body, cfg)
	if err != nil {
		return fmt.Errorf("signing failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Method:    %s\n", method)
	fmt.Fprintf(cmd.OutOrStdout(), "Signature: %s\n", sig)
	log.Debugf("content signed with method %s", method)
	return nil
}

func buildSignConfig() *maillib.MailSignatureConfig {
	var certChain []string
	if chain := viper.GetString("sign.cert-chain"); chain != "" {
		for _, f := range strings.Split(chain, ",") {
			if f = strings.TrimSpace(f); f != "" {
				certChain = append(certChain, f)
			}
		}
	}
	return &maillib.MailSignatureConfig{
		Method:           maillib.SigningMethod(viper.GetString("sign.method")),
		PrivateKeyFile:   viper.GetString("sign.private-key"),
		PublicKeyFile:    viper.GetString("sign.public-key"),
		KeyPassphrase:    viper.GetString("sign.passphrase"),
		CertificateChain: certChain,
		IncludeChain:     viper.GetBool("sign.include-chain"),
	}
}

func runVerify(cmd *cobra.Command, _ []string) error {
	method := viper.GetString("verify.method")
	body := viper.GetString("verify.body")
	signature := viper.GetString("verify.signature")

	if method == "" {
		return fmt.Errorf("signing method is required (--method)")
	}
	if body == "" {
		return fmt.Errorf("content is required (--body)")
	}
	if signature == "" {
		return fmt.Errorf("signature is required (--signature)")
	}
	if !maillib.IsValidSigningMethod(method) {
		return fmt.Errorf("invalid signing method %q: valid methods are %s",
			method, strings.Join(maillib.GetSupportedSigningMethods(), ", "))
	}

	cfg := &maillib.MailSignatureConfig{
		Method:         maillib.SigningMethod(method),
		PublicKeyFile:  viper.GetString("verify.public-key"),
		PrivateKeyFile: viper.GetString("verify.private-key"),
		KeyPassphrase:  viper.GetString("verify.passphrase"),
	}

	valid, err := maillib.VerifyMailSignature(body, signature, cfg)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	if valid {
		fmt.Fprintf(cmd.OutOrStdout(), "Signature valid (%s)\n", method)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Signature invalid (%s)\n", method)
	}
	return nil
}
