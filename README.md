# EmlToMarkdown
Drag an EML file into the application and it will convert to simple Markdown.

# Output
The converted Markdown is always written to a file rather than the terminal, since
copying out of a terminal is unreliable. Pass `--output-file` to choose the path
(it must not already exist); otherwise the output is written next to the input file
using its base name, so `abc 123.eml` becomes `abc 123.md`. An existing file is
never overwritten — a counter is appended instead (`abc 123_1.md`, `abc 123_2.md`,
...). If the input's directory cannot be written to, a temporary `.md` file in the
system temp directory is used. Once written, the file is opened in the OS's default
Markdown handler. If it cannot be opened automatically, the full path is printed so
you can open it yourself.

# Format
The EML file is parsed and Markdown is generated in the following format:

```
### EMAIL: [Subject Line Here]
* Date: YYYY-MM-DD HH:MM:SS UTC
* From: Jane Doe <jane@example.com>
* To: John Smith <john@example.com>
* Attachments (skipped):
  * Attachment 1 Name
  * Attachment 2 Name

[Email body text / formatted thread]
```

The `Date` is normalized to UTC. When a message has an HTML body it is converted
to Markdown; otherwise the plain-text body is used.

# Attachments
Attachment contents are not included in the output, but each attachment is listed
by name in the header under `Attachments (skipped)` so it is clear which files
were left out. When a message has no attachments, that section is omitted.

Image attachments are the exception: they are embedded in the output (see below)
rather than skipped, so they are not listed in the header.

# Images
Images are embedded directly into the Markdown as base64 data URIs so the output
is self-contained:

* Inline images embedded in the message (referenced by `cid:`) are embedded from
  the message itself.
* Images that point at a URL are fetched and embedded when they can be retrieved.
* If an image URL cannot be retrieved, a placeholder table containing
  "Failed to load image" is inserted in its place.
* Image attachments — and inline images the body never references — are appended
  to the end of the body, in the order they appear in the message, with the file
  name as the alt text.

Only image types a Markdown renderer can display are embedded (PNG, APNG, JPEG,
GIF, WebP, AVIF, BMP, SVG and ICO). Other image types (TIFF, HEIC, ...) are
treated as ordinary attachments and listed by name.
