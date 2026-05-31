// Package cmd commands
package cmd

import (
	"fmt"
	"os"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	cmdConfig     = "config"
	cmdConfigShow = "show"
	cmdConfigSave = "save"

	configNone = "(none)"

	// Repeated comment strings used in nodeComments.
	cAuthUsername  = "authentication username"
	cAuthPassword  = "authentication password"
	cUseStartTLS   = "use STARTTLS"
	cSkipTLSVerify = "skip TLS certificate verification"
	cSigningMethod = "signing method: rsa, ecdsa, gpg, smime"
	cPrivKeyFile   = "private key or S/MIME bundle PEM file"
	cPubKeyFile    = "public key or certificate file"
	cKeyPassphrase = "key passphrase"
	cCertChain     = "comma-separated cert chain PEM files (S/MIME)"
	cIncludeChain  = "include cert chain in S/MIME signature"
)

// skipConfigKeys are tool-internal keys that must not appear in a saved config.
var skipConfigKeys = map[string]bool{
	"config":    true, // would be a self-referential circular entry
	"unit-test": true, // test-only redirect flag
}

// nodeComments maps viper key paths to human-readable descriptions used as
// YAML comments in config show / config save output.
var nodeComments = map[string]string{
	// global
	"debug":    "verbose debug logging",
	"info":     "enable info-level logging",
	"no-color": "disable coloured log output",
	// smtp
	"smtp":                    "SMTP settings for outgoing mail (send command)",
	"smtp.server":             "hostname or IP of the SMTP server",
	"smtp.port":               "port: 25=plain, 465=SMTPS (ssl), 587=STARTTLS (tls)",
	"smtp.username":           cAuthUsername,
	"smtp.password":           cAuthPassword,
	"smtp.from":               "default sender address",
	"smtp.to":                 "default recipient(s)",
	"smtp.cc":                 "default CC recipient(s)",
	"smtp.bcc":                "default BCC recipient(s)",
	"smtp.subject":            "default subject line",
	"smtp.body":               "default body text",
	"smtp.attach":             "default attachment file(s), comma-separated",
	"smtp.ssl":                "use implicit TLS / SMTPS",
	"smtp.tls":                cUseStartTLS,
	"smtp.insecure":           cSkipTLSVerify,
	"smtp.auth":               "auth method: plain, login, crammd5, xoauth2",
	"smtp.html":               "send body as text/html",
	"smtp.timeout":            "connection timeout in seconds (0 = 15s default)",
	"smtp.helo":               "custom HELO/EHLO hostname",
	"smtp.max-size":           "max attachment size in bytes (0 = 5 MB default)",
	"smtp.sign":               "outgoing mail signing",
	"smtp.sign.method":        cSigningMethod,
	"smtp.sign.private-key":   cPrivKeyFile,
	"smtp.sign.public-key":    cPubKeyFile,
	"smtp.sign.passphrase":    cKeyPassphrase,
	"smtp.sign.cert-chain":    cCertChain,
	"smtp.sign.include-chain": cIncludeChain,
	// imap
	"imap":              "IMAP settings for reading mail (imap command)",
	"imap.server":       "hostname or IP of the IMAP server",
	"imap.port":         "port: 143=plain, 993=IMAPS (ssl)",
	"imap.username":     cAuthUsername,
	"imap.password":     cAuthPassword,
	"imap.inbox":        "default mailbox name",
	"imap.ssl":          "use implicit TLS / IMAPS",
	"imap.tls":          cUseStartTLS,
	"imap.insecure":     cSkipTLSVerify,
	"imap.timeout":      "connection timeout in seconds",
	"imap.download-dir": "directory to save attachments",
	// sign / verify (standalone commands)
	"sign":                "default settings for the sign command",
	viperSignMethod:       cSigningMethod,
	viperSignPrivateKey:   cPrivKeyFile,
	viperSignPublicKey:    cPubKeyFile,
	viperSignPassphrase:   cKeyPassphrase,
	viperSignCertChain:    cCertChain,
	viperSignIncludeChain: cIncludeChain,
	viperSignBody:         "content to sign",
	"verify":              "default settings for the verify command",
	viperVerifyMethod:     cSigningMethod,
	viperVerifyPublicKey:  cPubKeyFile,
	viperVerifyPrivateKey: "private key or S/MIME bundle fallback",
	viperVerifyPassphrase: cKeyPassphrase,
	viperVerifyBody:       "content to verify",
	viperVerifySignature:  "base64-encoded signature",
}

var (
	configOutput string

	configCmd = &cobra.Command{
		Use:   cmdConfig,
		Short: "Manage configuration",
		Long:  `Show and save the resolved mailcli configuration after all flags and environment variables are applied`,
	}

	configShowCmd = &cobra.Command{
		Use:          cmdConfigShow,
		Short:        "Print the active config file path and all resolved settings as valid YAML",
		RunE:         configShow,
		SilenceUsage: true,
	}

	configSaveCmd = &cobra.Command{
		Use:          cmdConfigSave,
		Short:        "Save resolved settings to a YAML config file",
		RunE:         configSave,
		SilenceUsage: true,
	}
)

func init() {
	configSaveCmd.Flags().StringVarP(&configOutput, "output", "o",
		configName+"."+configType, "output file path")
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSaveCmd)
	RootCmd.AddCommand(configCmd)
}

// effectiveSettings returns all viper settings with tool-internal keys removed.
func effectiveSettings() map[string]interface{} {
	all := viper.AllSettings()
	out := make(map[string]interface{}, len(all))
	for k, v := range all {
		if !skipConfigKeys[k] {
			out[k] = v
		}
	}
	return out
}

// nonZeroSettings recursively drops zero/empty/false values so that a saved
// config file contains only values that were explicitly configured.
func nonZeroSettings(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for k, v := range m {
		if sub, ok := v.(map[string]interface{}); ok {
			if cleaned := nonZeroSettings(sub); len(cleaned) > 0 {
				out[k] = cleaned
			}
			continue
		}
		rv := reflect.ValueOf(v)
		if !rv.IsValid() || rv.IsZero() {
			continue
		}
		if rv.Kind() == reflect.Slice && rv.Len() == 0 {
			continue
		}
		out[k] = v
	}
	return out
}

// annotateNodes walks a yaml.Node mapping tree and attaches comments from
// nodeComments. Section keys (whose values are mappings) get a HeadComment
// that appears on its own line before the key; scalar values get a
// LineComment that appears inline after the value.
func annotateNodes(node *yaml.Node, prefix string) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		path := keyNode.Value
		if prefix != "" {
			path = prefix + "." + keyNode.Value
		}
		if comment, ok := nodeComments[path]; ok {
			if valNode.Kind == yaml.MappingNode {
				keyNode.HeadComment = comment
			} else {
				valNode.LineComment = comment
			}
		}
		if valNode.Kind == yaml.MappingNode {
			annotateNodes(valNode, path)
		}
	}
}

// buildAnnotatedDoc marshals settings into a yaml.Node document, sets a
// header comment with the config file path and generation timestamp, and
// annotates all known keys with inline descriptions.
func buildAnnotatedDoc(settings map[string]interface{}, cfgUsed string) (*yaml.Node, error) {
	raw, err := yaml.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	var doc yaml.Node
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse config node: %w", err)
	}
	if len(doc.Content) > 0 {
		doc.Content[0].HeadComment = fmt.Sprintf(
			"mailcli configuration\n# Config file: %s\n# Generated:   %s",
			cfgUsed, time.Now().Format("2006-01-02 15:04:05"),
		)
		annotateNodes(doc.Content[0], "")
	}
	return &doc, nil
}

func configShow(cmd *cobra.Command, _ []string) error {
	cfgUsed := viper.ConfigFileUsed()
	if cfgUsed == "" {
		cfgUsed = configNone
	}
	doc, err := buildAnnotatedDoc(effectiveSettings(), cfgUsed)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("render config: %w", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s", out)
	return err
}

func configSave(cmd *cobra.Command, _ []string) error {
	outFile := configOutput
	if outFile == "" {
		outFile = configName + "." + configType
	}
	cfgUsed := viper.ConfigFileUsed()
	if cfgUsed == "" {
		cfgUsed = configNone
	}
	doc, err := buildAnnotatedDoc(nonZeroSettings(effectiveSettings()), cfgUsed)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("render config: %w", err)
	}
	if err = os.WriteFile(outFile, out, 0o600); err != nil {
		return fmt.Errorf("write config to %s: %w", outFile, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Config saved to %s\n", outFile)
	log.Debugf("config saved to %s", outFile)
	return nil
}
