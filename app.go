package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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

	content := Convert(input)

	if output == "" {
		fmt.Fprintln(stdout, content)
		fmt.Fprint(stdout, "PRESS ENTER TO EXIT")
		_, _ = bufio.NewReader(stdin).ReadString('\n')
		return 0
	}

	if err := os.WriteFile(output, []byte(content+"\n"), 0o644); err != nil {
		fmt.Fprintf(stderr, "Error writing output file: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote output to %s\n", output)
	return 0
}
