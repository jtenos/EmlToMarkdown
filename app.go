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

	// Without an explicit --output-file, write to a temporary .md file in the
	// system temp directory, since copying converted output out of a terminal is
	// unreliable.
	if output == "" {
		tmp, err := os.CreateTemp("", "emltomarkdown-*.md")
		if err != nil {
			fmt.Fprintf(stderr, "Error creating output file: %v\n", err)
			return 1
		}
		output = tmp.Name()
		tmp.Close()
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
