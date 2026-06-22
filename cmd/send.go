// Package cmd commands
package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tommi2day/gomodules/maillib"
	"github.com/wneessen/go-mail"
)

const cmdSend = "send"

var (
	smtpServer      = ""
	smtpPort        = 25
	smtpUsername    = ""
	smtpPassword    = ""
	smtpFrom        = ""
	smtpTo          []string
	smtpCC          []string
	smtpBCC         []string
	smtpSubject     = ""
	smtpBody        = ""
	smtpAttach      = ""
	smtpSSL         = false
	smtpTLS         = false
	smtpInsecure    = false
	smtpAuthMethod  = ""
	smtpContentHTML = false
	smtpTimeout     = int64(0)
	smtpHELO        = ""
	smtpMaxSize     = int64(0)

	smtpSignMethod     = ""
	smtpSignPrivKey    = ""
	smtpSignPubKey     = ""
	smtpSignPassphrase = ""
	smtpSignCertChain  = ""
	smtpSignIncChain   = false

	sendCmd = &cobra.Command{
		Use:          cmdSend + " [recipient...]",
		Short:        "Send an email via SMTP",
		Long:         `Send an email using SMTP with optional SSL/TLS and file attachments. Recipients can be given as positional arguments (mailx-style) in addition to or instead of --smtp.to.`,
		RunE:         sendMail,
		SilenceUsage: true,
	}
)

func init() {
	sendCmd.Flags().StringVarP(&smtpServer, "smtp.server", "S", "", "SMTP server hostname or IP")
	sendCmd.Flags().IntVarP(&smtpPort, "smtp.port", "P", 25, "SMTP server port")
	sendCmd.Flags().StringVarP(&smtpUsername, "smtp.username", "u", "", "SMTP authentication username")
	sendCmd.Flags().StringVarP(&smtpPassword, "smtp.password", "p", "", "SMTP authentication password")
	sendCmd.Flags().StringVarP(&smtpFrom, "smtp.from", "f", "", "sender email address")
	sendCmd.Flags().StringSliceVarP(&smtpTo, "smtp.to", "t", nil, "recipient address(es); comma-separated or repeated flag; also accepted as positional args")
	sendCmd.Flags().StringSliceVarP(&smtpCC, "smtp.cc", "c", nil, "CC recipient(s); comma-separated or repeated flag")
	sendCmd.Flags().StringSliceVarP(&smtpBCC, "smtp.bcc", "b", nil, "BCC recipient(s); comma-separated or repeated flag")
	sendCmd.Flags().StringVarP(&smtpSubject, "smtp.subject", "s", "", "email subject (required)")
	sendCmd.Flags().StringVar(&smtpBody, "smtp.body", "", "email body text (omit to read from stdin)")
	sendCmd.Flags().StringVarP(&smtpAttach, "smtp.attach", "a", "", "attachment file(s), comma-separated paths")
	sendCmd.Flags().BoolVar(&smtpSSL, "smtp.ssl", false, "use SMTPS (port 465)")
	sendCmd.Flags().BoolVar(&smtpTLS, "smtp.tls", false, "use STARTTLS (port 587)")
	sendCmd.Flags().BoolVar(&smtpInsecure, "smtp.insecure", false, "skip TLS/SSL certificate verification")
	sendCmd.Flags().StringVar(&smtpAuthMethod, "smtp.auth", "", "SMTP auth method (plain/login/crammd5/xoauth2/none); defaults to plain when username+password are set")
	sendCmd.Flags().BoolVar(&smtpContentHTML, "smtp.html", false, "send body as HTML")
	sendCmd.Flags().Int64VarP(&smtpTimeout, "smtp.timeout", "T", 0, "connection timeout in seconds (0=default 15s)")
	sendCmd.Flags().StringVar(&smtpHELO, "smtp.helo", "", "custom HELO hostname")
	sendCmd.Flags().Int64Var(&smtpMaxSize, "smtp.max-size", 0, "max attachment size in bytes (0=5MB default)")
	sendCmd.Flags().StringVar(&smtpSignMethod, "smtp.sign.method", "", "signing method for outgoing mail: rsa, ecdsa, gpg, smime")
	sendCmd.Flags().StringVar(&smtpSignPrivKey, "smtp.sign.private-key", "", "private key or S/MIME certificate bundle file")
	sendCmd.Flags().StringVar(&smtpSignPubKey, "smtp.sign.public-key", "", "public key or certificate file (S/MIME)")
	sendCmd.Flags().StringVar(&smtpSignPassphrase, "smtp.sign.passphrase", "", "key passphrase")
	sendCmd.Flags().StringVar(&smtpSignCertChain, "smtp.sign.cert-chain", "", "comma-separated cert chain PEM files (S/MIME)")
	sendCmd.Flags().BoolVar(&smtpSignIncChain, "smtp.sign.include-chain", false, "include cert chain in S/MIME signature")

	if err := viper.BindPFlags(sendCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	RootCmd.AddCommand(sendCmd)
}

func sendMail(cmd *cobra.Command, args []string) error {
	server := viper.GetString("smtp.server")
	port := viper.GetInt("smtp.port")
	subject := viper.GetString("smtp.subject")

	// Merge --smtp.to values with positional arguments (mailx-style).
	toList := append(viper.GetStringSlice("smtp.to"), args...)

	if port == 0 {
		port = 25
	}
	if server == "" {
		return fmt.Errorf("SMTP server is required (--smtp.server or smtp.server in config)")
	}
	if len(toList) == 0 {
		return fmt.Errorf("recipient is required (--smtp.to, positional args, or smtp.to in config)")
	}
	if subject == "" {
		return fmt.Errorf("subject is required (--smtp.subject or smtp.subject in config)")
	}
	authMethod := viper.GetString("smtp.auth")
	if authMethod != "" && authMethod != "none" && viper.GetString("smtp.password") == "" {
		return fmt.Errorf("smtp.password is required when smtp.auth is set (use smtp.auth=none for anonymous send)")
	}

	log.Debugf("send: connecting to %s:%d", server, port)

	to := strings.Join(toList, ",")
	cc := strings.Join(viper.GetStringSlice("smtp.cc"), ",")
	bcc := strings.Join(viper.GetStringSlice("smtp.bcc"), ",")
	config := buildSMTPConfig(server, port, viper.GetString("smtp.username"), viper.GetString("smtp.password"))
	mailObj := buildMailObject(viper.GetString("smtp.from"), to, cc, bcc, viper.GetString("smtp.attach"))
	body := viper.GetString("smtp.body")
	if body == "" {
		body = readStdinBody(cmd)
	}

	if err := applySignature(mailObj, &body); err != nil {
		return err
	}
	if err := config.SendMail(mailObj, subject, body); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	log.Info("Mail sent successfully")
	return nil
}

// readStdinBody reads the body from stdin when it is piped (not a terminal).
// If cmd has a custom reader injected (e.g. in tests), it reads unconditionally.
func readStdinBody(cmd *cobra.Command) string {
	in := cmd.InOrStdin()
	if in == os.Stdin {
		fi, err := os.Stdin.Stat()
		if err != nil || (fi.Mode()&os.ModeCharDevice) != 0 {
			return ""
		}
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\r\n")
}

func buildSMTPConfig(server string, port int, username, password string) *maillib.SendMailConfigType {
	config := maillib.NewSendMailConfig(server, port, username, password)
	insecure := viper.GetBool("smtp.insecure")
	if viper.GetBool("smtp.ssl") {
		config.ServerConfig.EnableSSL(insecure)
	} else if viper.GetBool("smtp.tls") {
		config.ServerConfig.EnableTLS(insecure)
	}
	auth := viper.GetString("smtp.auth")
	if auth == "" && username != "" && password != "" {
		auth = "plain"
	}
	if auth != "" && auth != "none" {
		config.ServerConfig.SetAuthMethod(auth)
	}
	if timeout := viper.GetInt64("smtp.timeout"); timeout > 0 {
		config.ServerConfig.SetTimeout(timeout)
	}
	if helo := viper.GetString("smtp.helo"); helo != "" {
		config.ServerConfig.SetHELO(helo)
	}
	if maxSize := viper.GetInt64("smtp.max-size"); maxSize > 0 {
		config.SetMaxSize(maxSize)
	}
	if viper.GetBool("smtp.html") {
		config.SetContentType(mail.TypeTextHTML)
	}
	return config
}

// applySignature configures signing on outgoing mail.
// For S/MIME it sets SignatureConfig (handled by SendMail).
// For RSA/ECDSA/GPG it signs the body and appends the signature block.
func applySignature(mailObj *maillib.MailType, body *string) error {
	method := viper.GetString("smtp.sign.method")
	if method == "" {
		return nil
	}
	if !maillib.IsValidSigningMethod(method) {
		return fmt.Errorf("invalid signing method %q: valid methods are %s",
			method, strings.Join(maillib.GetSupportedSigningMethods(), ", "))
	}
	cfg := buildSendSignatureConfig()
	if cfg.Method == maillib.SigningMethodSMIME {
		mailObj.SignatureConfig = cfg
		return nil
	}
	sig, err := maillib.SignMailContent(*body, cfg)
	if err != nil {
		return fmt.Errorf("signing failed: %w", err)
	}
	*body = *body + "\n\n--- " + method + " Signature ---\n" + sig
	return nil
}

func buildSendSignatureConfig() *maillib.MailSignatureConfig {
	var certChain []string
	if chain := viper.GetString("smtp.sign.cert-chain"); chain != "" {
		for _, f := range strings.Split(chain, ",") {
			if f = strings.TrimSpace(f); f != "" {
				certChain = append(certChain, f)
			}
		}
	}
	return &maillib.MailSignatureConfig{
		Method:           maillib.SigningMethod(viper.GetString("smtp.sign.method")),
		PrivateKeyFile:   viper.GetString("smtp.sign.private-key"),
		PublicKeyFile:    viper.GetString("smtp.sign.public-key"),
		KeyPassphrase:    viper.GetString("smtp.sign.passphrase"),
		CertificateChain: certChain,
		IncludeChain:     viper.GetBool("smtp.sign.include-chain"),
	}
}

func buildMailObject(from, to, cc, bcc, attach string) *maillib.MailType {
	mailObj := maillib.NewMail(from, to)
	if cc != "" {
		mailObj.SetCc(cc)
	}
	if bcc != "" {
		mailObj.SetBcc(bcc)
	}
	if attach != "" {
		parts := strings.Split(strings.TrimSpace(attach), ",")
		trimmed := make([]string, len(parts))
		for i, f := range parts {
			trimmed[i] = strings.TrimSpace(f)
		}
		mailObj.SetAttach(trimmed)
	}
	return mailObj
}
