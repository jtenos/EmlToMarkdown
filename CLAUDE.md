# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

A single-binary Go CLI that converts an `.eml` email file into a self-contained Markdown file, then opens it in the OS's default Markdown handler. Aimed at drag-and-drop use: dragging an `.eml` onto the binary passes it as a lone argument.

## Commands

All common tasks are wrapped in the `Makefile`:

```bash
make build          # go build -o emltomarkdown .
make build-win-x64  # cross-compile emltomarkdown.exe (GOOS=windows GOARCH=amd64)
make test           # go test ./...
make run ARGS="--input-file mail.eml"   # go run . <args>
make fmt            # go fmt ./...
make vet            # go vet ./...
make clean          # go clean and remove built binaries
```

Run a single test with the standard toolchain, e.g.:

```bash
go test -run TestConvert -v ./...
```

## CLI behavior

- Input is resolved in priority order: `--input-file`, then a single unnamed `.eml` argument (drag-and-drop), then an interactive stdin prompt.
- `--output-file` is optional and **must not already exist**; this is checked before any conversion work. Without it, `defaultOutputFile` writes next to the input file as `<input base name>.md`, appending `_1`, `_2`, ... when that name is taken (created with `O_EXCL`, so an existing file is never overwritten); if the input's directory is not writable it falls back to a temp `*.md` file in the system temp dir. Terminal copy-paste is deemed unreliable, so output is never printed to stdout.
- After writing, the file is opened in the OS default handler; on failure the path is printed instead.

## Architecture

The conversion pipeline is `Convert` → `walkPart` → `buildBody`/`buildHeader`, with clear seams for testing:

- **[main.go](main.go)** — thin entrypoint; delegates to `run` so all I/O (args, stdin, stdout, stderr) is injectable.
- **[app.go](app.go)** — `run` handles all flag/input/output-file orchestration and exit codes. Kept separate from `main` specifically so parameter handling is unit-testable ([app_test.go](app_test.go)).
- **[convert.go](convert.go)** — `Convert` (file path) wraps `convert` (arbitrary `io.Reader` + injected `imageFetcher`). Builds the fixed header block (`### EMAIL:`, Date normalized to UTC, From/To normalized to `Name <addr>`, optional `CC`/`BCC` lines emitted only when those headers are present and non-empty, optional `Attachments (skipped)` list). Handles RFC 2047 encoded-word decoding.
- **[mime.go](mime.go)** — `walkPart` recursively flattens the MIME tree into `mailParts` (first `text/html`, first `text/plain`, inline images keyed by Content-ID, every embeddable image in document order, and attachment names in document order). Any image whose media type is in `markdownImageTypes` (`isMarkdownImage`) goes into `parts.images` regardless of disposition — image attachments are embedded rather than listed as skipped; other image types fall through to the attachment list. `decodeBody` reverses Content-Transfer-Encoding (base64 with whitespace stripped, quoted-printable). Note: `mime/multipart` transparently decodes quoted-printable parts, so those pass through already decoded.
- **[body.go](body.go)** — `buildBody` prefers the HTML part (falling back to plain text), then appends every image from `parts.images` the body did not already display — `rewriteImage` records resolved Content-IDs in the `used` set so a `cid:` image is not repeated at the bottom. `htmlToMarkdown` rewrites every `<img>` to be self-contained before conversion: `cid:` refs resolved from inline images, `http(s)` fetched and inlined, all as base64 data URIs. Unresolvable images become a `| Failed to load image |` GFM table via a placeholder-token swap done *after* Markdown conversion (so the table markup survives HTML escaping). HTML→Markdown uses `JohannesKaufmann/html-to-markdown/v2` with base + commonmark + table + strikethrough plugins.
- **[open.go](open.go)** — `openMarkdownFile` (a package var so tests can stub it) selects the launch command per platform. **WSL is detected specially** (reads `/proc/version` for "microsoft"/"wsl") and prefers `wslview`, falling back to `cmd.exe /c start`; otherwise `open` (macOS), `xdg-open` (Linux), `start` (Windows).

### Testability seams to preserve

- `imageFetcher` is injected through `convert` so tests run without network access (`defaultFetch` is the real HTTP implementation with a 15s timeout and 25 MB cap).
- `openMarkdownFile` is a package-level variable for stubbing.
- `run` takes explicit readers/writers rather than touching `os.*` directly.

Test fixtures live in [testdata/](testdata/): `plain.eml`, `html_inline.eml`, `html_remote.eml`, `image_attachment.eml`.
