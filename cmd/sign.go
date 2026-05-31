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

const (
	cmdSign   = "sign"
	cmdVerify = "verify"

	flagMethod     = "method"
	flagPrivateKey = "private-key"
	flagPublicKey  = "public-key"
	flagBody       = "body"
	flagPassphrase = "passphrase"
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
		Use:          cmdSign,
		Short:        "Sign content and print the signature",
		RunE:         runSign,
		SilenceUsage: true,
	}

	verifyCmd = &cobra.Command{
		Use:          cmdVerify,
		Short:        "Verify a mail content signature",
		RunE:         runVerify,
		SilenceUsage: true,
	}
)

func init() {
	signCmd.Flags().StringVarP(&signMethod, flagMethod, "m", "", "signing method: rsa, ecdsa, gpg, smime (required)")
	signCmd.Flags().StringVar(&signPrivKey, flagPrivateKey, "", "private key or S/MIME certificate bundle file (required)")
	signCmd.Flags().StringVar(&signPubKey, flagPublicKey, "", "public key or certificate file")
	signCmd.Flags().StringVarP(&signBody, flagBody, "b", "", "content to sign (required)")
	signCmd.Flags().StringVar(&signPassphrase, flagPassphrase, "", "key passphrase")
	signCmd.Flags().StringVar(&signCertChain, "cert-chain", "", "comma-separated cert chain PEM files (S/MIME)")
	signCmd.Flags().BoolVar(&signIncChain, "include-chain", false, "include cert chain in S/MIME signature")
	bindNamedFlags(signCmd, map[string]string{
		"sign.method":        flagMethod,
		"sign.private-key":   flagPrivateKey,
		"sign.public-key":    flagPublicKey,
		"sign.body":          flagBody,
		"sign.passphrase":    flagPassphrase,
		"sign.cert-chain":    "cert-chain",
		"sign.include-chain": "include-chain",
	})

	verifyCmd.Flags().StringVarP(&verifyMethod, flagMethod, "m", "", "signing method: rsa, ecdsa, gpg, smime (required)")
	verifyCmd.Flags().StringVar(&verifyPubKey, flagPublicKey, "", "public key or certificate file")
	verifyCmd.Flags().StringVar(&verifyPrivKey, flagPrivateKey, "", "certificate bundle (S/MIME fallback when no public-key)")
	verifyCmd.Flags().StringVarP(&verifyBody, flagBody, "b", "", "content to verify (required)")
	verifyCmd.Flags().StringVar(&verifyPassphrase, flagPassphrase, "", "key passphrase")
	verifyCmd.Flags().StringVar(&verifySignature, "signature", "", "base64-encoded signature to verify (required)")
	bindNamedFlags(verifyCmd, map[string]string{
		"verify.method":      flagMethod,
		"verify.public-key":  flagPublicKey,
		"verify.private-key": flagPrivateKey,
		"verify.body":        flagBody,
		"verify.passphrase":  flagPassphrase,
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
