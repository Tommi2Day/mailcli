// Package cmd commands
package cmd

import (
	"fmt"
	"strings"

	goimap "github.com/emersion/go-imap"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tommi2day/gomodules/maillib"
)

const defaultInbox = "INBOX"

var (
	imapServer        = ""
	imapPort          = 143
	imapUsername      = ""
	imapPassword      = ""
	imapInbox         = defaultInbox
	imapSSL           = false
	imapTLS           = false
	imapInsecure      = false
	imapTimeout       = int64(0)
	imapDownloadDir   = "."
	imapSearchText    = ""
	imapMessageIDs    = ""
	imapSaveAttach    = false
	imapVerify        = false
	imapVerifyPubKey  = ""
	imapVerifyPrivKey = ""
	imapVerifyPass    = ""

	imapCmd = &cobra.Command{
		Use:   "imap",
		Short: "IMAP mail commands",
		Long:  `Commands for reading and managing emails via IMAP`,
	}

	imapListCmd = &cobra.Command{
		Use:          "list",
		Short:        "List available mailboxes",
		RunE:         imapListMailboxes,
		SilenceUsage: true,
	}

	imapStatusCmd = &cobra.Command{
		Use:          "status",
		Short:        "Show mailbox status (total and unseen count)",
		RunE:         imapMailboxStatus,
		SilenceUsage: true,
	}

	imapReadCmd = &cobra.Command{
		Use:          "read",
		Short:        "Read messages from mailbox (unseen by default)",
		RunE:         imapReadMessages,
		SilenceUsage: true,
	}

	imapDeleteCmd = &cobra.Command{
		Use:          "delete",
		Short:        "Permanently delete messages by sequence IDs",
		RunE:         imapDeleteMessages,
		SilenceUsage: true,
	}

	imapSearchCmd = &cobra.Command{
		Use:          "search",
		Short:        "Search messages in mailbox",
		RunE:         imapSearch,
		SilenceUsage: true,
	}
)

func init() {
	imapCmd.PersistentFlags().StringVarP(&imapServer, "imap.server", "s", "", "IMAP server hostname or IP")
	imapCmd.PersistentFlags().IntVarP(&imapPort, "imap.port", "p", 143, "IMAP server port")
	imapCmd.PersistentFlags().StringVarP(&imapUsername, "imap.username", "u", "", "IMAP username")
	imapCmd.PersistentFlags().StringVarP(&imapPassword, "imap.password", "P", "", "IMAP password")
	imapCmd.PersistentFlags().StringVarP(&imapInbox, "imap.inbox", "i", defaultInbox, "mailbox/folder to use")
	imapCmd.PersistentFlags().BoolVar(&imapSSL, "imap.ssl", false, "use IMAPS (SSL, port 993)")
	imapCmd.PersistentFlags().BoolVar(&imapTLS, "imap.tls", false, "use STARTTLS")
	imapCmd.PersistentFlags().BoolVar(&imapInsecure, "imap.insecure", false, "skip TLS/SSL certificate verification")
	imapCmd.PersistentFlags().Int64Var(&imapTimeout, "imap.timeout", 0, "connection timeout in seconds")
	imapCmd.PersistentFlags().StringVarP(&imapDownloadDir, "imap.download-dir", "d", ".", "directory to save attachments")

	if err := viper.BindPFlags(imapCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}

	imapReadCmd.Flags().StringVar(&imapSearchText, "text", "", "only show messages containing this text")
	imapReadCmd.Flags().BoolVar(&imapSaveAttach, "save-attachments", false, "save attachments to download-dir")
	imapReadCmd.Flags().BoolVar(&imapVerify, "verify-signature", false, "verify message signatures if present")
	imapReadCmd.Flags().StringVar(&imapVerifyPubKey, "verify-public-key", "", "public key or certificate for signature verification")
	imapReadCmd.Flags().StringVar(&imapVerifyPrivKey, "verify-private-key", "", "private key or S/MIME bundle for signature verification")
	imapReadCmd.Flags().StringVar(&imapVerifyPass, "verify-passphrase", "", "passphrase for the verify key")

	imapSearchCmd.Flags().StringVarP(&imapSearchText, "text", "T", "", "search text in message body")

	imapDeleteCmd.Flags().StringVar(&imapMessageIDs, "ids", "", "comma-separated message sequence IDs to delete (required)")

	imapCmd.AddCommand(imapListCmd)
	imapCmd.AddCommand(imapStatusCmd)
	imapCmd.AddCommand(imapReadCmd)
	imapCmd.AddCommand(imapDeleteCmd)
	imapCmd.AddCommand(imapSearchCmd)
	RootCmd.AddCommand(imapCmd)
}

func buildImapConfig() (*maillib.ImapType, error) {
	server := viper.GetString("imap.server")
	port := viper.GetInt("imap.port")
	username := viper.GetString("imap.username")
	password := viper.GetString("imap.password")
	inbox := viper.GetString("imap.inbox")
	useSSL := viper.GetBool("imap.ssl")
	useTLS := viper.GetBool("imap.tls")
	insecure := viper.GetBool("imap.insecure")
	timeout := viper.GetInt64("imap.timeout")
	downloadDir := viper.GetString("imap.download-dir")

	if port == 0 {
		port = 143
	}
	if inbox == "" {
		inbox = defaultInbox
	}
	if downloadDir == "" {
		downloadDir = "."
	}

	if server == "" {
		return nil, fmt.Errorf("IMAP server is required (--imap.server or imap.server in config)")
	}

	config := maillib.NewImapConfig(server, port, username, password)
	config.Inbox = inbox
	config.SetDownloadDir(downloadDir)

	if useSSL {
		config.ServerConfig.EnableSSL(insecure)
	} else if useTLS {
		config.ServerConfig.EnableTLS(insecure)
	}
	if timeout > 0 {
		config.ServerConfig.SetTimeout(timeout)
	}
	return config, nil
}

func imapListMailboxes(cmd *cobra.Command, _ []string) error {
	config, err := buildImapConfig()
	if err != nil {
		return err
	}
	if err = config.Connect(); err != nil {
		return fmt.Errorf("IMAP connect failed: %w", err)
	}
	defer config.LogOut()

	mailboxes, err := config.ListMailboxes()
	if err != nil {
		return fmt.Errorf("list mailboxes failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Mailboxes on %s:\n", viper.GetString("imap.server"))
	for _, mb := range mailboxes {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", mb)
	}
	return nil
}

func imapMailboxStatus(cmd *cobra.Command, _ []string) error {
	config, err := buildImapConfig()
	if err != nil {
		return err
	}
	if err = config.Connect(); err != nil {
		return fmt.Errorf("IMAP connect failed: %w", err)
	}
	defer config.LogOut()

	inbox := viper.GetString("imap.inbox")
	if inbox == "" {
		inbox = defaultInbox
	}
	all, unseen, flags, err := config.MBoxStatus(inbox)
	if err != nil {
		return fmt.Errorf("mailbox status failed: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Mailbox: %s\n", inbox)
	fmt.Fprintf(cmd.OutOrStdout(), "  Total:  %d\n", all)
	fmt.Fprintf(cmd.OutOrStdout(), "  Unseen: %d\n", unseen)
	fmt.Fprintf(cmd.OutOrStdout(), "  Flags:  %s\n", strings.Join(flags, ", "))
	return nil
}

func imapReadMessages(cmd *cobra.Command, _ []string) error {
	config, err := buildImapConfig()
	if err != nil {
		return err
	}
	if err = config.Connect(); err != nil {
		return fmt.Errorf("IMAP connect failed: %w", err)
	}
	defer config.LogOut()

	var ids []uint32
	if imapSearchText != "" {
		criteria := goimap.NewSearchCriteria()
		criteria.Text = []string{imapSearchText}
		ids, err = config.SearchMessages(criteria)
	} else {
		ids, err = config.GetUnseenMessageIDs()
	}
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(ids) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No messages found")
		return nil
	}

	log.Debugf("reading %d messages", len(ids))
	msgs, err := config.ReadMessages(ids)
	if err != nil {
		return fmt.Errorf("read messages failed: %w", err)
	}

	for _, msg := range msgs {
		parsed, parseErr := config.ParseMessage(msg, imapSaveAttach)
		if parseErr != nil {
			log.Warnf("parse message %d failed: %s", msg.UID, parseErr)
			continue
		}
		printParsedMessage(cmd, parsed)
		if imapVerify {
			verifyParsedSignature(cmd, parsed)
		}
	}
	return nil
}

func printParsedMessage(cmd *cobra.Command, parsed maillib.MailType) {
	fmt.Fprintf(cmd.OutOrStdout(), "--- Message #%d ---\n", parsed.ID)
	fmt.Fprintf(cmd.OutOrStdout(), "From:    %s\n", parsed.From)
	fmt.Fprintf(cmd.OutOrStdout(), "To:      %s\n", strings.Join(parsed.To, ", "))
	if len(parsed.CC) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "CC:      %s\n", strings.Join(parsed.CC, ", "))
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Subject: %s\n", parsed.Subject)
	fmt.Fprintf(cmd.OutOrStdout(), "Date:    %s\n", parsed.Date.Format("2006-01-02 15:04:05"))
	if len(parsed.Attachments) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Attach:  %s\n", strings.Join(parsed.Attachments, ", "))
	}
	for i, part := range parsed.TextParts {
		fmt.Fprintf(cmd.OutOrStdout(), "--- Body (part %d) ---\n%s\n", i+1, part)
	}
}

// extractSignatureBlock parses an inline signature block appended by applySignature in send.go.
// Format: "<body>\n\n--- <method> Signature ---\n<base64sig>"
func extractSignatureBlock(text string) (method, originalBody, sig string, ok bool) {
	const marker = "\n\n--- "
	const suffix = " Signature ---\n"
	idx := strings.Index(text, marker)
	if idx < 0 {
		return
	}
	after := text[idx+len(marker):]
	suffIdx := strings.Index(after, suffix)
	if suffIdx < 0 {
		return
	}
	method = after[:suffIdx]
	sig = strings.TrimSpace(after[suffIdx+len(suffix):])
	originalBody = text[:idx]
	ok = true
	return
}

func verifyParsedSignature(cmd *cobra.Command, parsed maillib.MailType) {
	for _, att := range parsed.Attachments {
		if strings.HasSuffix(strings.ToLower(att), "smime.p7s") {
			fmt.Fprintf(cmd.OutOrStdout(), "Signature-Method: smime\nSignature-Status: detected (S/MIME verification requires raw MIME body)\n")
			return
		}
	}
	for _, part := range parsed.TextParts {
		method, body, sig, ok := extractSignatureBlock(part)
		if !ok {
			continue
		}
		if !maillib.IsValidSigningMethod(method) {
			fmt.Fprintf(cmd.OutOrStdout(), "Signature-Method: %s\nSignature-Status: unknown signing method\n", method)
			return
		}
		cfg := &maillib.MailSignatureConfig{
			Method:         maillib.SigningMethod(method),
			PublicKeyFile:  imapVerifyPubKey,
			PrivateKeyFile: imapVerifyPrivKey,
			KeyPassphrase:  imapVerifyPass,
		}
		valid, err := maillib.VerifyMailSignature(body, sig, cfg)
		switch {
		case err != nil:
			fmt.Fprintf(cmd.OutOrStdout(), "Signature-Method: %s\nSignature-Status: error (%s)\n", method, err)
		case valid:
			fmt.Fprintf(cmd.OutOrStdout(), "Signature-Method: %s\nSignature-Status: valid\n", method)
		default:
			fmt.Fprintf(cmd.OutOrStdout(), "Signature-Method: %s\nSignature-Status: invalid\n", method)
		}
		return
	}
}

func imapDeleteMessages(cmd *cobra.Command, _ []string) error {
	config, err := buildImapConfig()
	if err != nil {
		return err
	}
	if imapMessageIDs == "" {
		return fmt.Errorf("message IDs required (--ids)")
	}
	if err = config.Connect(); err != nil {
		return fmt.Errorf("IMAP connect failed: %w", err)
	}
	defer config.LogOut()

	var ids []uint32
	for _, idStr := range strings.Split(imapMessageIDs, ",") {
		idStr = strings.TrimSpace(idStr)
		var id uint32
		if _, scanErr := fmt.Sscanf(idStr, "%d", &id); scanErr != nil {
			return fmt.Errorf("invalid message ID %q: %w", idStr, scanErr)
		}
		ids = append(ids, id)
	}

	if err = config.PurgeMessages(ids); err != nil {
		return fmt.Errorf("delete messages failed: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Deleted %d message(s)\n", len(ids))
	return nil
}

func imapSearch(cmd *cobra.Command, _ []string) error {
	config, err := buildImapConfig()
	if err != nil {
		return err
	}
	if err = config.Connect(); err != nil {
		return fmt.Errorf("IMAP connect failed: %w", err)
	}
	defer config.LogOut()

	criteria := goimap.NewSearchCriteria()
	if imapSearchText != "" {
		criteria.Text = []string{imapSearchText}
	} else {
		criteria.WithoutFlags = []string{goimap.SeenFlag}
	}

	ids, err := config.SearchMessages(criteria)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(ids) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No messages found")
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Found %d message(s): %v\n", len(ids), ids)
	return nil
}
