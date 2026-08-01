# EmlToMarkdown
Drag an EML file into the application and it will convert to simple Markdown.

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

# Images
Images in the body are embedded directly into the Markdown as base64 data URIs so
the output is self-contained:

* Inline images embedded in the message (referenced by `cid:`) are embedded from
  the message itself.
* Images that point at a URL are fetched and embedded when they can be retrieved.
* If an image URL cannot be retrieved, a placeholder table containing
  "Failed to load image" is inserted in its place.
