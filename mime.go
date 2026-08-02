package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
)

// inlineImage holds the decoded bytes and media type of an embedded image so it
// can be turned into a data URI when building the body.
type inlineImage struct {
	name        string // file name, when the part supplied one
	contentID   string // Content-ID without angle brackets, when the part supplied one
	contentType string
	data        []byte
}

// mailParts is the flattened, decoded view of a MIME message that the body and
// header builders work from.
type mailParts struct {
	html        []byte                 // first text/html body found
	text        []byte                 // first text/plain body found
	inline      map[string]inlineImage // Content-ID (without angle brackets) -> image
	images      []inlineImage          // embeddable images, in document order
	attachments []string               // attachment file names, in document order
}

// markdownImageTypes are the image media types a Markdown renderer can display
// from a data URI, since it renders images through an HTML <img>. Image types
// outside this set (image/tiff, image/heic, ...) are treated as ordinary
// attachments and listed by name instead.
var markdownImageTypes = map[string]bool{
	"image/png":                true,
	"image/apng":               true,
	"image/jpeg":               true,
	"image/jpg":                true, // non-standard, but seen in the wild
	"image/gif":                true,
	"image/webp":               true,
	"image/avif":               true,
	"image/bmp":                true,
	"image/x-bmp":              true,
	"image/x-ms-bmp":           true,
	"image/svg+xml":            true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
}

// isMarkdownImage reports whether a media type can be embedded in Markdown as a
// base64 data URI.
func isMarkdownImage(mediaType string) bool {
	return markdownImageTypes[strings.ToLower(strings.TrimSpace(mediaType))]
}

// walkPart recursively descends a MIME tree, decoding each leaf and sorting it
// into the appropriate bucket of parts. header/body are the current part's
// header and (undecoded, framing-stripped) body reader.
func walkPart(header textproto.MIMEHeader, body io.Reader, parts *mailParts) error {
	ctype := header.Get("Content-Type")
	if ctype == "" {
		ctype = "text/plain"
	}
	mediaType, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		mediaType, params = "text/plain", nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if err := walkPart(p.Header, p, parts); err != nil {
				return err
			}
		}
	}

	data, err := decodeBody(header, body)
	if err != nil {
		return err
	}

	disposition, dparams, _ := mime.ParseMediaType(header.Get("Content-Disposition"))
	filename := dparams["filename"]
	if filename == "" {
		filename = params["name"]
	}
	filename = decodeWord(filename)
	contentID := strings.Trim(strings.TrimSpace(header.Get("Content-ID")), "<>")
	isAttachment := strings.EqualFold(disposition, "attachment")

	isImage := strings.HasPrefix(mediaType, "image/")

	switch {
	case isImage && isMarkdownImage(mediaType):
		// An image we can embed, whether it arrived inline or as an attachment.
		// A Content-ID lets the HTML body reference it with cid:; it is also
		// queued for the end of the body, and buildBody drops the ones the body
		// already displayed.
		img := inlineImage{name: filename, contentID: contentID, contentType: mediaType, data: data}
		if contentID != "" {
			parts.inline[contentID] = img
		}
		parts.images = append(parts.images, img)
	case isImage && !isAttachment && contentID != "":
		// An image type Markdown cannot reliably render, but the body points at
		// it by cid:, so embed it anyway and let the renderer decide.
		parts.inline[contentID] = inlineImage{contentType: mediaType, data: data}
	case isAttachment:
		parts.attachments = append(parts.attachments, attachmentName(filename, mediaType))
	case mediaType == "text/html" && parts.html == nil:
		parts.html = data
	case mediaType == "text/plain" && parts.text == nil:
		parts.text = data
	default:
		// Anything else that is not part of the readable body (a nested
		// message, an unreferenced binary part, etc.) is reported by name.
		parts.attachments = append(parts.attachments, attachmentName(filename, mediaType))
	}
	return nil
}

// attachmentName returns a display name for an attachment, falling back to the
// media type when no file name was provided.
func attachmentName(filename, mediaType string) string {
	if filename != "" {
		return filename
	}
	return "(" + mediaType + ")"
}

// decodeBody reads the part body and reverses its Content-Transfer-Encoding.
//
// Note that mime/multipart transparently decodes quoted-printable parts and
// hides the header, so those arrive here already decoded and pass through.
func decodeBody(header textproto.MIMEHeader, body io.Reader) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(header.Get("Content-Transfer-Encoding"))) {
	case "base64":
		raw, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		// Email base64 is wrapped across many lines; strip whitespace before
		// decoding so the strict decoder does not choke on it.
		cleaned := bytes.Map(func(r rune) rune {
			if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
				return -1
			}
			return r
		}, raw)
		return base64.StdEncoding.DecodeString(string(cleaned))
	case "quoted-printable":
		return io.ReadAll(quotedprintable.NewReader(body))
	default:
		return io.ReadAll(body)
	}
}
