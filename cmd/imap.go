// Package cmd commands
package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	goimap "github.com/emersion/go-imap"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tommi2day/gomodules/maillib"
)

const defaultInbox = "INBOX"

const (
	cmdImap   = "imap"
	cmdList   = "list"
	cmdStatus = "status"
	cmdRead   = "read"
	cmdDelete = "delete"
	cmdSearch = "search"
)

const (
	securityPlain     = "  "
	securitySigned    = "S "
	securityEncrypted = " E"
	securityBoth      = "SE"
)

type listEntry struct {
	seqNum   uint32
	date     string
	security string
	from     string
	subject  string
}

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
	imapSearchQuery   = ""
	imapMessageIDs    = ""
	imapSaveAttach    = false
	imapVerify        = false
	imapVerifyPubKey  = ""
	imapVerifyPrivKey = ""
	imapVerifyPass    = ""
	imapListFolders   = false
	imapListAll       = false

	imapCmd = &cobra.Command{
		Use:   cmdImap,
		Short: "IMAP mail commands",
		Long:  `Commands for reading and managing emails via IMAP`,
	}

	imapListCmd = &cobra.Command{
		Use:          cmdList,
		Short:        "List messages in mailbox (unseen by default); use --folders to list mailboxes",
		RunE:         imapList,
		SilenceUsage: true,
	}

	imapStatusCmd = &cobra.Command{
		Use:          cmdStatus,
		Short:        "Show mailbox status (total and unseen count)",
		RunE:         imapMailboxStatus,
		SilenceUsage: true,
	}

	imapReadCmd = &cobra.Command{
		Use:          cmdRead,
		Short:        "Read messages from mailbox (unseen by default)",
		RunE:         imapReadMessages,
		SilenceUsage: true,
	}

	imapDeleteCmd = &cobra.Command{
		Use:          cmdDelete,
		Short:        "Permanently delete messages by sequence IDs",
		RunE:         imapDeleteMessages,
		SilenceUsage: true,
	}

	imapSearchCmd = &cobra.Command{
		Use:          cmdSearch,
		Short:        "Search messages in mailbox",
		RunE:         imapSearch,
		SilenceUsage: true,
	}
)

func init() {
	imapCmd.PersistentFlags().StringVarP(&imapServer, "imap.server", "S", "", "IMAP server hostname or IP")
	imapCmd.PersistentFlags().IntVarP(&imapPort, "imap.port", "P", 143, "IMAP server port")
	imapCmd.PersistentFlags().StringVarP(&imapUsername, "imap.username", "u", "", "IMAP username")
	imapCmd.PersistentFlags().StringVarP(&imapPassword, "imap.password", "p", "", "IMAP password")
	imapCmd.PersistentFlags().StringVarP(&imapInbox, "imap.inbox", "i", defaultInbox, "mailbox/folder to use")
	imapCmd.PersistentFlags().BoolVar(&imapSSL, "imap.ssl", false, "use IMAPS (SSL, port 993)")
	imapCmd.PersistentFlags().BoolVar(&imapTLS, "imap.tls", false, "use STARTTLS")
	imapCmd.PersistentFlags().BoolVar(&imapInsecure, "imap.insecure", false, "skip TLS/SSL certificate verification")
	imapCmd.PersistentFlags().Int64VarP(&imapTimeout, "imap.timeout", "T", 0, "connection timeout in seconds")
	imapCmd.PersistentFlags().StringVarP(&imapDownloadDir, "imap.download-dir", "d", ".", "directory to save attachments")

	if err := viper.BindPFlags(imapCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}

	imapListCmd.Flags().BoolVar(&imapListFolders, "folders", false, "list mailboxes instead of messages")
	imapListCmd.Flags().BoolVar(&imapListAll, "all", false, "list all messages (default: unseen only)")

	imapReadCmd.Flags().StringVar(&imapMessageIDs, "ids", "", "comma-separated message sequence IDs to read")
	imapReadCmd.Flags().StringVar(&imapSearchQuery, "query", "", "show only messages matching this string in any header or body (IMAP TEXT)")
	imapReadCmd.Flags().BoolVar(&imapSaveAttach, "save-attachments", false, "save attachments to download-dir")
	imapReadCmd.Flags().BoolVar(&imapVerify, "verify-signature", false, "verify message signatures if present")
	imapReadCmd.Flags().StringVar(&imapVerifyPubKey, "verify-public-key", "", "public key or certificate for signature verification")
	imapReadCmd.Flags().StringVar(&imapVerifyPrivKey, "verify-private-key", "", "private key or S/MIME bundle for signature verification")
	imapReadCmd.Flags().StringVar(&imapVerifyPass, "verify-passphrase", "", "passphrase for the verify key")

	imapSearchCmd.Flags().StringVar(&imapSearchQuery, "query", "", "search string matched against any header or body (IMAP TEXT)")

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

func imapList(cmd *cobra.Command, _ []string) error {
	config, err := buildImapConfig()
	if err != nil {
		return err
	}
	if err = config.Connect(); err != nil {
		return fmt.Errorf("IMAP connect failed: %w", err)
	}
	defer config.LogOut()

	if imapListFolders {
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

	return imapListMessages(cmd, config)
}

func imapListMessages(cmd *cobra.Command, config *maillib.ImapType) error {
	inbox := viper.GetString("imap.inbox")
	if inbox == "" {
		inbox = defaultInbox
	}

	seqset, empty, err := buildListSeqSet(config, inbox)
	if err != nil {
		return err
	}
	if empty {
		label := "unseen "
		if imapListAll {
			label = ""
		}
		fmt.Fprintf(cmd.OutOrStdout(), "No %smessages in %s\n", label, inbox)
		return nil
	}

	entries, err := fetchListEntries(config, seqset)
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].seqNum < entries[j].seqNum })

	fmt.Fprintf(cmd.OutOrStdout(), "Messages in %s (%d listed):\n", inbox, len(entries))
	for _, e := range entries {
		fmt.Fprintf(cmd.OutOrStdout(), "  #%-4d  %-16s  %2s  %-35s  %s\n",
			e.seqNum, e.date, e.security, truncate(e.from, 35), e.subject,
		)
	}
	return nil
}

// buildListSeqSet returns the sequence set to fetch and whether the mailbox is
// empty. Selects the mailbox as a side-effect (required before Fetch).
func buildListSeqSet(config *maillib.ImapType, inbox string) (*goimap.SeqSet, bool, error) {
	if imapListAll {
		total, _, _, err := config.MBoxStatus(inbox)
		if err != nil {
			return nil, false, fmt.Errorf("list messages failed: %w", err)
		}
		if total == 0 {
			return nil, true, nil
		}
		seqset, err := goimap.ParseSeqSet("1:*")
		if err != nil {
			return nil, false, fmt.Errorf("build seqset failed: %w", err)
		}
		return seqset, false, nil
	}
	ids, err := config.GetUnseenMessageIDs()
	if err != nil {
		return nil, false, fmt.Errorf("list messages failed: %w", err)
	}
	if len(ids) == 0 {
		return nil, true, nil
	}
	seqset := new(goimap.SeqSet)
	seqset.AddNum(ids...)
	return seqset, false, nil
}

// fetchListEntries fetches ENVELOPE + Content-Type for the given sequence set
// and returns one listEntry per message. No full body is downloaded.
//
// Fetch is run in a goroutine so the main goroutine can drain msgChan
// concurrently; without this, a mailbox with more messages than the channel
// buffer causes a deadlock (go-imap's reader goroutine blocks on send while
// Fetch blocks waiting for the response to finish).
func fetchListEntries(config *maillib.ImapType, seqset *goimap.SeqSet) ([]listEntry, error) {
	ctSection := &goimap.BodySectionName{
		BodyPartName: goimap.BodyPartName{
			Specifier: goimap.HeaderSpecifier,
			Fields:    []string{"Content-Type"},
		},
		Peek: true,
	}
	msgChan := make(chan *goimap.Message)
	items := []goimap.FetchItem{goimap.FetchEnvelope, goimap.FetchUid, ctSection.FetchItem()}

	fetchErr := make(chan error, 1)
	go func() {
		fetchErr <- config.Client.Fetch(seqset, items, msgChan)
	}()

	var entries []listEntry
	for msg := range msgChan {
		entries = append(entries, parseListEntry(msg, ctSection))
	}
	if err := <-fetchErr; err != nil {
		return nil, fmt.Errorf("fetch message headers failed: %w", err)
	}
	return entries, nil
}

func parseListEntry(msg *goimap.Message, ctSection *goimap.BodySectionName) listEntry {
	e := listEntry{seqNum: msg.SeqNum, security: securityPlain}
	if env := msg.Envelope; env != nil {
		e.date = env.Date.Format("2006-01-02 15:04")
		e.subject = env.Subject
		e.from = formatSenderAddress(env.From)
	}
	if r := msg.GetBody(ctSection); r != nil {
		if data, readErr := io.ReadAll(r); readErr == nil {
			e.security = contentTypeSecurityStatus(string(data))
		}
	}
	return e
}

func formatSenderAddress(addrs []*goimap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	a := addrs[0]
	if a.PersonalName != "" {
		return a.PersonalName + " <" + a.MailboxName + "@" + a.HostName + ">"
	}
	return a.MailboxName + "@" + a.HostName
}

// contentTypeSecurityStatus inspects a raw Content-Type header block and returns
// a 2-char indicator: S=signed, E=encrypted, SE=both, "  "=plain.
// Detects S/MIME (pkcs7) and PGP/MIME (RFC 3156) from the Content-Type alone,
// without downloading the message body.
func contentTypeSecurityStatus(headerBlock string) string {
	var ct string
	for _, line := range strings.Split(headerBlock, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "content-type:") {
			ct = strings.ToLower(strings.TrimSpace(line[len("content-type:"):]))
			break
		}
	}
	if ct == "" {
		return securityPlain
	}

	var signed, encrypted bool
	switch {
	case strings.HasPrefix(ct, "multipart/signed"):
		signed = true
	case strings.HasPrefix(ct, "multipart/encrypted"):
		encrypted = true
	case strings.HasPrefix(ct, "application/pkcs7-mime"):
		if strings.Contains(ct, "enveloped-data") {
			encrypted = true
		} else {
			signed = true
		}
	case strings.HasPrefix(ct, "application/pgp-encrypted"):
		encrypted = true
	}

	switch {
	case signed && encrypted:
		return securityBoth
	case signed:
		return securitySigned
	case encrypted:
		return securityEncrypted
	default:
		return securityPlain
	}
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return s[:limit-3] + "..."
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
	switch {
	case imapMessageIDs != "":
		ids, err = parseMessageIDs(imapMessageIDs)
	case imapSearchQuery != "":
		criteria := goimap.NewSearchCriteria()
		criteria.Text = []string{imapSearchQuery}
		ids, err = config.SearchMessages(criteria)
	default:
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

func parseMessageIDs(raw string) ([]uint32, error) {
	var ids []uint32
	for _, idStr := range strings.Split(raw, ",") {
		idStr = strings.TrimSpace(idStr)
		var id uint32
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			return nil, fmt.Errorf("invalid message ID %q: %w", idStr, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
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

	ids, err := parseMessageIDs(imapMessageIDs)
	if err != nil {
		return err
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
	if imapSearchQuery != "" {
		criteria.Text = []string{imapSearchQuery}
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
