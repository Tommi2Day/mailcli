// Package cmd commands
package cmd

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tommi2day/gomodules/maillib"
	"github.com/wneessen/go-mail"
)

var (
	smtpServer      = ""
	smtpPort        = 25
	smtpUsername    = ""
	smtpPassword    = ""
	smtpFrom        = ""
	smtpTo          = ""
	smtpCC          = ""
	smtpBCC         = ""
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

	sendCmd = &cobra.Command{
		Use:          "send",
		Short:        "Send an email via SMTP",
		Long:         `Send an email using SMTP with optional SSL/TLS and file attachments`,
		RunE:         sendMail,
		SilenceUsage: true,
	}
)

func init() {
	sendCmd.Flags().StringVarP(&smtpServer, "smtp.server", "s", "", "SMTP server hostname or IP")
	sendCmd.Flags().IntVarP(&smtpPort, "smtp.port", "p", 25, "SMTP server port")
	sendCmd.Flags().StringVarP(&smtpUsername, "smtp.username", "u", "", "SMTP authentication username")
	sendCmd.Flags().StringVarP(&smtpPassword, "smtp.password", "P", "", "SMTP authentication password")
	sendCmd.Flags().StringVarP(&smtpFrom, "smtp.from", "f", "", "sender email address")
	sendCmd.Flags().StringVarP(&smtpTo, "smtp.to", "t", "", "recipient address(es), comma-separated (required)")
	sendCmd.Flags().StringVar(&smtpCC, "smtp.cc", "", "CC recipient(s), comma-separated")
	sendCmd.Flags().StringVar(&smtpBCC, "smtp.bcc", "", "BCC recipient(s), comma-separated")
	sendCmd.Flags().StringVarP(&smtpSubject, "smtp.subject", "S", "", "email subject (required)")
	sendCmd.Flags().StringVarP(&smtpBody, "smtp.body", "b", "", "email body text")
	sendCmd.Flags().StringVarP(&smtpAttach, "smtp.attach", "a", "", "attachment file(s), comma-separated paths")
	sendCmd.Flags().BoolVar(&smtpSSL, "smtp.ssl", false, "use SMTPS (port 465)")
	sendCmd.Flags().BoolVar(&smtpTLS, "smtp.tls", false, "use STARTTLS (port 587)")
	sendCmd.Flags().BoolVar(&smtpInsecure, "smtp.insecure", false, "skip TLS/SSL certificate verification")
	sendCmd.Flags().StringVar(&smtpAuthMethod, "smtp.auth", "", "SMTP auth method (plain/login/crammd5/xoauth2)")
	sendCmd.Flags().BoolVar(&smtpContentHTML, "smtp.html", false, "send body as HTML")
	sendCmd.Flags().Int64Var(&smtpTimeout, "smtp.timeout", 0, "connection timeout in seconds (0=default 15s)")
	sendCmd.Flags().StringVar(&smtpHELO, "smtp.helo", "", "custom HELO hostname")
	sendCmd.Flags().Int64Var(&smtpMaxSize, "smtp.max-size", 0, "max attachment size in bytes (0=5MB default)")

	if err := viper.BindPFlags(sendCmd.Flags()); err != nil {
		log.Fatal(err)
	}
	RootCmd.AddCommand(sendCmd)
}

func sendMail(cmd *cobra.Command, _ []string) error {
	server := viper.GetString("smtp.server")
	port := viper.GetInt("smtp.port")
	to := viper.GetString("smtp.to")
	subject := viper.GetString("smtp.subject")

	if port == 0 {
		port = 25
	}
	if server == "" {
		return fmt.Errorf("SMTP server is required (--smtp.server or smtp.server in config)")
	}
	if to == "" {
		return fmt.Errorf("recipient is required (--smtp.to or smtp.to in config)")
	}
	if subject == "" {
		return fmt.Errorf("subject is required (--smtp.subject or smtp.subject in config)")
	}

	log.Debugf("send: connecting to %s:%d", server, port)

	config := buildSMTPConfig(server, port, viper.GetString("smtp.username"), viper.GetString("smtp.password"))
	mailObj := buildMailObject(viper.GetString("smtp.from"), to, viper.GetString("smtp.cc"), viper.GetString("smtp.bcc"), viper.GetString("smtp.attach"))

	if err := config.SendMail(mailObj, subject, viper.GetString("smtp.body")); err != nil {
		return fmt.Errorf("send failed: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Mail sent successfully")
	return nil
}

func buildSMTPConfig(server string, port int, username, password string) *maillib.SendMailConfigType {
	config := maillib.NewSendMailConfig(server, port, username, password)
	insecure := viper.GetBool("smtp.insecure")
	if viper.GetBool("smtp.ssl") {
		config.ServerConfig.EnableSSL(insecure)
	} else if viper.GetBool("smtp.tls") {
		config.ServerConfig.EnableTLS(insecure)
	}
	if auth := viper.GetString("smtp.auth"); auth != "" {
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
