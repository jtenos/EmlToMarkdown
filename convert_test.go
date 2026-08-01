package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// stubFetch stands in for network access: any URL containing "good" resolves to
// a fake image, everything else fails.
func stubFetch(url string) (string, []byte, error) {
	if strings.Contains(url, "good") {
		return "image/gif", []byte("GIF89a-fake-bytes"), nil
	}
	return "", nil, fmt.Errorf("not found")
}

// convertFixture parses a testdata EML file with the stub fetcher.
func convertFixture(t *testing.T, name string) string {
	t.Helper()
	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatalf("opening fixture: %v", err)
	}
	defer f.Close()
	out, err := convert(f, stubFetch)
	if err != nil {
		t.Fatalf("convert(%s): %v", name, err)
	}
	return out
}

func TestConvert_PlainText(t *testing.T) {
	want := "### EMAIL: Lunch tomorrow?\n" +
		"* Date: 2025-07-14 13:30:00 UTC\n" +
		"* From: Jane Doe <jane@example.com>\n" +
		"* To: John Smith <john@example.com>\n" +
		"\n" +
		"Hi John,\n\n" +
		"Are you free for lunch tomorrow? I was thinking we could try that new place downtown.\n\n" +
		"Cheers,\nJane"

	if got := convertFixture(t, "plain.eml"); got != want {
		t.Errorf("output mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestConvert_HTMLInlineImageAndAttachment(t *testing.T) {
	got := convertFixture(t, "html_inline.eml")

	checks := []string{
		"### EMAIL: Quarterly Report 📊",                                // RFC 2047 encoded-word subject decoded
		"* From: Doe, Jane <jane@example.com>",                         // quoted display name
		"* To: John Smith <john@example.com>, Team <team@example.com>", // address list
		"* Attachments (skipped):",
		"  * report.pdf",
		"**quarterly**",                               // HTML converted to Markdown
		"![Company Logo](data:image/png;base64,iVBOR", // inline cid: image embedded
	}
	for _, c := range checks {
		if !strings.Contains(got, c) {
			t.Errorf("output missing %q:\n%s", c, got)
		}
	}
	// The embedded image must not be listed as an attachment.
	if strings.Contains(got, "logo.png") {
		t.Errorf("embedded inline image should not appear as an attachment:\n%s", got)
	}
}

func TestConvert_RemoteImages(t *testing.T) {
	got := convertFixture(t, "html_remote.eml")

	if !strings.Contains(got, "![Banner](data:image/gif;base64,") {
		t.Errorf("reachable remote image should be embedded as a data URI:\n%s", got)
	}
	if !strings.Contains(got, failedImageMarkdown) {
		t.Errorf("unreachable remote image should be replaced by the failed-image table:\n%s", got)
	}
}

func TestConvert_NoAttachmentsSectionWhenNone(t *testing.T) {
	if got := convertFixture(t, "plain.eml"); strings.Contains(got, "Attachments") {
		t.Errorf("plain email should have no attachments section:\n%s", got)
	}
}

func TestConvert_FileNotFound(t *testing.T) {
	if _, err := Convert("testdata/does-not-exist.eml"); err == nil {
		t.Fatal("expected an error for a missing input file, got nil")
	}
}
