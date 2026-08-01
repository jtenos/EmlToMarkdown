package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureEML is a real, parseable EML file used to exercise the parameter
// handling in run without stubbing the conversion.
const fixtureEML = "testdata/plain.eml"

// fixtureSubject is a line that must appear in the converted output of fixtureEML.
const fixtureSubject = "### EMAIL: Lunch tomorrow?"

// runHelper invokes run with the given args and stdin, capturing stdout/stderr.
func runHelper(t *testing.T, args []string, stdin string) (code int, stdout, stderr string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	code = run(args, strings.NewReader(stdin), &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

func TestRun_OutputToFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "result.md")

	code, stdout, stderr := runHelper(t, []string{"--input-file", fixtureEML, "--output-file", out}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("output file %q should contain the converted subject line", string(data))
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Errorf("output file should end with a trailing newline: %q", string(data))
	}
	if !strings.Contains(stdout, out) {
		t.Errorf("stdout %q should mention output path %q", stdout, out)
	}
}

func TestRun_OutputFileAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(out, []byte("do not overwrite"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runHelper(t, []string{"--input-file", fixtureEML, "--output-file", out}, "")

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero for existing output file")
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("stderr %q should report that the file already exists", stderr)
	}
	// The pre-existing file must be left untouched.
	data, _ := os.ReadFile(out)
	if string(data) != "do not overwrite" {
		t.Errorf("existing file was modified: %q", string(data))
	}
}

func TestRun_OutputToScreen(t *testing.T) {
	code, stdout, stderr := runHelper(t, []string{"--input-file", fixtureEML}, "\n")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, fixtureSubject) {
		t.Errorf("stdout %q should contain the converted content", stdout)
	}
	if !strings.Contains(stdout, "PRESS ENTER TO EXIT") {
		t.Errorf("stdout %q should contain the PRESS ENTER TO EXIT prompt", stdout)
	}
}

func TestRun_PromptsForInputFile(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "result.md")

	// No --input-file, so the app should prompt and read the path from stdin.
	code, stdout, stderr := runHelper(t, []string{"--output-file", out}, fixtureEML+"\n")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "Enter the path") {
		t.Errorf("stdout %q should contain the input prompt", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("output file %q should contain the converted subject line", string(data))
	}
}

func TestRun_PositionalEmlArgumentTreatedAsInput(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "result.md")

	// A single unnamed .eml argument should be used as the input file,
	// with no prompt for input. (Go's flag package requires flags to precede
	// positional arguments.)
	code, stdout, stderr := runHelper(t, []string{"--output-file", out, fixtureEML}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if strings.Contains(stdout, "Enter the path") {
		t.Errorf("stdout %q should not prompt when a positional .eml file is given", stdout)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("output file %q should contain the converted subject line", string(data))
	}
}

func TestRun_LonePositionalEmlArgument(t *testing.T) {
	// The bare drag-and-drop case: the executable is invoked with just the
	// .eml path and outputs to the screen.
	code, stdout, stderr := runHelper(t, []string{fixtureEML}, "\n")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, fixtureSubject) {
		t.Errorf("stdout %q should contain the converted content", stdout)
	}
}

func TestRun_PositionalNonEmlArgumentIgnored(t *testing.T) {
	// A single unnamed argument that is not a .eml file should not be treated
	// as the input file; the app falls back to prompting.
	code, stdout, _ := runHelper(t, []string{"notanemail.txt"}, "\n")

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero when no valid input file is provided")
	}
	if !strings.Contains(stdout, "Enter the path") {
		t.Errorf("stdout %q should prompt when the positional arg is not a .eml file", stdout)
	}
}

func TestRun_NoInputFileProvided(t *testing.T) {
	// Not passed on the command line and nothing entered at the prompt.
	code, _, stderr := runHelper(t, []string{}, "\n")

	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero when no input file is provided")
	}
	if !strings.Contains(stderr, "no input file") {
		t.Errorf("stderr %q should report the missing input file", stderr)
	}
}

func TestRun_InputFileDoesNotExist(t *testing.T) {
	// A named input file that cannot be opened should fail with a non-zero code.
	code, _, stderr := runHelper(t, []string{"--input-file", "testdata/nope.eml"}, "")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero for a missing input file")
	}
	if !strings.Contains(stderr, "converting") {
		t.Errorf("stderr %q should report the conversion error", stderr)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	code, _, _ := runHelper(t, []string{"--nonsense"}, "")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero for an unknown flag")
	}
}
