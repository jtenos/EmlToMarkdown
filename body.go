package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// imageFetcher retrieves a remote image, returning its media type and bytes. It
// is injected so tests can run without network access.
type imageFetcher func(url string) (contentType string, data []byte, err error)

// failedImageMarkdown is the 1x1 GFM table shown in place of an image whose URL
// could not be fetched.
const failedImageMarkdown = "| Failed to load image |\n| --- |"

// buildBody renders the readable message body as Markdown, preferring the HTML
// part (with images embedded) and falling back to plain text.
func buildBody(parts *mailParts, fetch imageFetcher) (string, error) {
	if len(parts.html) > 0 {
		return htmlToMarkdown(parts.html, parts.inline, fetch)
	}
	return strings.TrimRight(string(parts.text), " \t\r\n"), nil
}

// htmlToMarkdown resolves image references in the HTML (inline cid: images and
// remote URLs become data URIs; unreachable URLs become a placeholder table)
// and converts the result to Markdown.
func htmlToMarkdown(htmlBody []byte, inline map[string]inlineImage, fetch imageFetcher) (string, error) {
	doc, err := html.Parse(strings.NewReader(string(htmlBody)))
	if err != nil {
		return "", err
	}

	// Placeholder tokens are swapped for the failed-image table after
	// conversion, so the exact table markup survives HTML escaping.
	var placeholders []string
	resolveImages(doc, inline, fetch, &placeholders)

	md, err := newConverter().ConvertNode(doc)
	if err != nil {
		return "", err
	}
	out := string(md)
	for _, token := range placeholders {
		out = strings.ReplaceAll(out, token, failedImageMarkdown)
	}
	return strings.TrimRight(out, " \t\r\n"), nil
}

// newConverter builds a Markdown converter tuned for email HTML: standard
// CommonMark plus GFM tables and strikethrough.
func newConverter() *converter.Converter {
	return converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
			strikethrough.NewStrikethroughPlugin(),
		),
	)
}

// resolveImages walks the parsed HTML and rewrites every <img> src so the
// output is self-contained.
func resolveImages(n *html.Node, inline map[string]inlineImage, fetch imageFetcher, placeholders *[]string) {
	if n.Type == html.ElementNode && n.DataAtom == atom.Img {
		rewriteImage(n, inline, fetch, placeholders)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		resolveImages(c, inline, fetch, placeholders)
	}
}

// rewriteImage replaces an <img> node's src with an embedded data URI, or turns
// the node into a placeholder token when the source cannot be resolved.
func rewriteImage(n *html.Node, inline map[string]inlineImage, fetch imageFetcher, placeholders *[]string) {
	src := strings.TrimSpace(getAttr(n, "src"))
	switch {
	case src == "" || strings.HasPrefix(src, "data:"):
		// Nothing to do: empty or already embedded.
		return
	case strings.HasPrefix(strings.ToLower(src), "cid:"):
		id := strings.TrimSpace(src[len("cid:"):])
		if img, ok := inline[id]; ok {
			setAttr(n, "src", dataURI(img.contentType, img.data))
			return
		}
		replaceWithPlaceholder(n, placeholders)
	case strings.HasPrefix(strings.ToLower(src), "http://"), strings.HasPrefix(strings.ToLower(src), "https://"):
		contentType, data, err := fetch(src)
		if err != nil || len(data) == 0 {
			replaceWithPlaceholder(n, placeholders)
			return
		}
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		setAttr(n, "src", dataURI(contentType, data))
	default:
		// Unknown scheme we cannot embed (file:, etc.).
		replaceWithPlaceholder(n, placeholders)
	}
}

// replaceWithPlaceholder swaps the node for a unique text token that is turned
// into the failed-image table once conversion is complete.
func replaceWithPlaceholder(n *html.Node, placeholders *[]string) {
	token := fmt.Sprintf("EMLFAILEDIMAGE%dPLACEHOLDER", len(*placeholders))
	*placeholders = append(*placeholders, token)
	text := &html.Node{Type: html.TextNode, Data: token}
	if p := n.Parent; p != nil {
		p.InsertBefore(text, n)
		p.RemoveChild(n)
	}
}

// dataURI encodes bytes as a base64 data URI of the given media type.
func dataURI(contentType string, data []byte) string {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

// defaultFetch retrieves a remote image over HTTP with a short timeout and
// treats any non-2xx response as a failure.
func defaultFetch(url string) (string, []byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 25<<20))
	if err != nil {
		return "", nil, err
	}
	return resp.Header.Get("Content-Type"), data, nil
}
