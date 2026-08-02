package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// run executes the application logic and returns the process exit code.
//
// It is separated from main so that all input/output can be injected, making
// the parameter-handling behaviour straightforward to unit test.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("emltomarkdown", flag.ContinueOnError)
	fs.SetOutput(stderr)
	inputFile := fs.String("input-file", "", "The EML file to parse")
	outputFile := fs.String("output-file", "", "The MD file to output")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	input := strings.TrimSpace(*inputFile)
	if input == "" {
		// Support drag-and-drop: a single unnamed .eml argument is treated
		// as the input file.
		if rest := fs.Args(); len(rest) == 1 && strings.EqualFold(filepath.Ext(rest[0]), ".eml") {
			input = strings.TrimSpace(rest[0])
		}
	}
	if input == "" {
		fmt.Fprint(stdout, "Enter the path to the EML file to parse: ")
		line, err := bufio.NewReader(stdin).ReadString('\n')
		input = strings.TrimSpace(line)
		if input == "" {
			if err != nil && !errors.Is(err, io.EOF) {
				fmt.Fprintf(stderr, "Error reading input file path: %v\n", err)
			} else {
				fmt.Fprintln(stderr, "Error: no input file provided")
			}
			return 1
		}
	}

	// An explicit --output-file must not already exist. Check this before doing
	// any conversion work so the failure is reported promptly.
	output := strings.TrimSpace(*outputFile)
	if output != "" {
		if _, err := os.Stat(output); err == nil {
			fmt.Fprintf(stderr, "Error: output file %q already exists\n", output)
			return 1
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "Error checking output file: %v\n", err)
			return 1
		}
	}

	content, err := Convert(input)
	if err != nil {
		fmt.Fprintf(stderr, "Error converting %s: %v\n", input, err)
		return 1
	}

	// Without an explicit --output-file, write alongside the input file, since
	// copying converted output out of a terminal is unreliable.
	if output == "" {
		output, err = defaultOutputFile(input)
		if err != nil {
			// The input's directory may not be writable (read-only media, a
			// mounted archive, ...), so fall back to the system temp directory.
			tmp, tmpErr := os.CreateTemp("", "emltomarkdown-*.md")
			if tmpErr != nil {
				fmt.Fprintf(stderr, "Error creating output file: %v\n", err)
				return 1
			}
			output = tmp.Name()
			tmp.Close()
		}
	}

	if err := os.WriteFile(output, []byte(content+"\n"), 0o644); err != nil {
		fmt.Fprintf(stderr, "Error writing output file: %v\n", err)
		return 1
	}

	// Try to open the result in the OS's default Markdown handler. If that fails,
	// print the path so the user can open it themselves.
	if err := openMarkdownFile(output); err != nil {
		fmt.Fprintf(stderr, "Error opening output file: %v\n", err)
		fmt.Fprintf(stdout, "Wrote output to %s\n", output)
	} else {
		fmt.Fprintf(stdout, "Opened %s\n", output)
	}
	return 0
}

// maxOutputSuffix bounds the counter appended to the default output file name so
// a directory full of matching names cannot spin forever.
const maxOutputSuffix = 1000

// defaultOutputFile reserves a new .md file next to the input file, named after
// it: mail.eml becomes mail.md. An existing file is never overwritten; a counter
// is appended instead (mail_1.md, mail_2.md, ...). The file is created empty so
// the name cannot be taken between the check and the later write.
func defaultOutputFile(input string) (string, error) {
	dir := filepath.Dir(input)
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	for i := 0; i <= maxOutputSuffix; i++ {
		name := base + ".md"
		if i > 0 {
			name = fmt.Sprintf("%s_%d.md", base, i)
		}
		candidate := filepath.Join(dir, name)
		f, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			f.Close()
			return candidate, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("no available output file name for %s after %d attempts", input, maxOutputSuffix)
}
