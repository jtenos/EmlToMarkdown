package main

import (
	"fmt"
	"io"
	"mime"
	"net/mail"
	"net/textproto"
	"os"
	"strings"
)

// Convert parses the EML file at inputPath and returns the resulting Markdown.
func Convert(inputPath string) (string, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return convert(f, defaultFetch)
}

// convert does the real work of Convert against an arbitrary reader, with the
// image fetcher injected so it can be exercised without network access.
func convert(r io.Reader, fetch imageFetcher) (string, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return "", fmt.Errorf("parsing message: %w", err)
	}

	parts := &mailParts{inline: map[string]inlineImage{}}
	if err := walkPart(textproto.MIMEHeader(msg.Header), msg.Body, parts); err != nil {
		return "", fmt.Errorf("reading message body: %w", err)
	}

	body, err := buildBody(parts, fetch)
	if err != nil {
		return "", fmt.Errorf("building body: %w", err)
	}

	return buildHeader(msg.Header, parts.attachments) + body, nil
}

// buildHeader renders the fixed metadata block that precedes the message body.
func buildHeader(h mail.Header, attachments []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "### EMAIL: %s\n", decodeWord(h.Get("Subject")))
	fmt.Fprintf(&b, "* Date: %s\n", formatDate(h))
	fmt.Fprintf(&b, "* From: %s\n", formatAddresses(h.Get("From")))
	fmt.Fprintf(&b, "* To: %s\n", formatAddresses(h.Get("To")))
	// Cc/Bcc are usually absent, so they are only listed when present.
	if cc := formatAddresses(h.Get("Cc")); cc != "" {
		fmt.Fprintf(&b, "* CC: %s\n", cc)
	}
	if bcc := formatAddresses(h.Get("Bcc")); bcc != "" {
		fmt.Fprintf(&b, "* BCC: %s\n", bcc)
	}
	if len(attachments) > 0 {
		b.WriteString("* Attachments (skipped):\n")
		for _, name := range attachments {
			fmt.Fprintf(&b, "  * %s\n", name)
		}
	}
	b.WriteString("\n")
	return b.String()
}

// formatDate renders the message Date header as "YYYY-MM-DD HH:MM:SS UTC",
// falling back to the raw header value when it cannot be parsed.
func formatDate(h mail.Header) string {
	t, err := h.Date()
	if err != nil {
		return strings.TrimSpace(h.Get("Date"))
	}
	return t.UTC().Format("2006-01-02 15:04:05") + " UTC"
}

// formatAddresses normalises an address-list header to "Name <addr>" form,
// falling back to the decoded raw value when it cannot be parsed.
func formatAddresses(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	addrs, err := mail.ParseAddressList(raw)
	if err != nil {
		return decodeWord(raw)
	}
	formatted := make([]string, len(addrs))
	for i, a := range addrs {
		if a.Name != "" {
			formatted[i] = fmt.Sprintf("%s <%s>", a.Name, a.Address)
		} else {
			formatted[i] = fmt.Sprintf("<%s>", a.Address)
		}
	}
	return strings.Join(formatted, ", ")
}

// decodeWord decodes RFC 2047 encoded-word syntax (e.g. "=?UTF-8?B?...?=") in a
// header value, returning the input unchanged when it is not encoded.
func decodeWord(s string) string {
	decoded, err := (&mime.WordDecoder{}).DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}
