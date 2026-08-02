package main

import (
	"bytes"
	"errors"
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

// lastOpened records the path passed to the stubbed file opener during a test.
var lastOpened string

// TestMain replaces the real OS file opener with a stub so tests never launch a
// GUI application, and records the path each test opened.
func TestMain(m *testing.M) {
	openMarkdownFile = func(path string) error {
		lastOpened = path
		return nil
	}
	os.Exit(m.Run())
}

// runHelper invokes run with the given args and stdin, capturing stdout/stderr.
func runHelper(t *testing.T, args []string, stdin string) (code int, stdout, stderr string) {
	t.Helper()
	lastOpened = ""
	var outBuf, errBuf bytes.Buffer
	code = run(args, strings.NewReader(stdin), &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// copyFixture copies fixtureEML into a fresh temp directory under the given
// name, so tests that exercise the default output location write next to a
// throwaway input file rather than polluting testdata/.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(fixtureEML)
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing fixture copy: %v", err)
	}
	return path
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
	if lastOpened != out {
		t.Errorf("opened %q, want the output file %q", lastOpened, out)
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

func TestRun_DefaultOutputNextToInputFile(t *testing.T) {
	// With no --output-file, the app writes "<input base name>.md" in the input
	// file's own directory and opens it. The name may contain spaces.
	input := copyFixture(t, "abc 123.eml")
	want := filepath.Join(filepath.Dir(input), "abc 123.md")

	code, stdout, stderr := runHelper(t, []string{"--input-file", input}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if lastOpened != want {
		t.Errorf("opened %q, want %q", lastOpened, want)
	}
	if !strings.Contains(stdout, want) {
		t.Errorf("stdout %q should mention the output path %q", stdout, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("output file %q should contain the converted subject line", string(data))
	}
}

func TestRun_DefaultOutputAddsCounterWhenNameTaken(t *testing.T) {
	// Existing default-named files are never overwritten; a counter is appended.
	input := copyFixture(t, "abc 123.eml")
	dir := filepath.Dir(input)
	for _, name := range []string{"abc 123.md", "abc 123_1.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("do not overwrite"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	code, _, stderr := runHelper(t, []string{"--input-file", input}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	want := filepath.Join(dir, "abc 123_2.md")
	if lastOpened != want {
		t.Errorf("opened %q, want %q", lastOpened, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("output file %q should contain the converted subject line", string(data))
	}
	// The pre-existing files must be left untouched.
	for _, name := range []string{"abc 123.md", "abc 123_1.md"} {
		existing, _ := os.ReadFile(filepath.Join(dir, name))
		if string(existing) != "do not overwrite" {
			t.Errorf("existing file %s was modified: %q", name, string(existing))
		}
	}
}

func TestRun_DefaultOutputFallsBackToTempDir(t *testing.T) {
	// When the input's directory cannot take the output file, the app falls back
	// to a temporary file rather than failing.
	input := copyFixture(t, "readonly.eml")
	dir := filepath.Dir(input)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	code, stdout, stderr := runHelper(t, []string{"--input-file", input}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if lastOpened == "" {
		t.Fatal("expected a temporary output file to be opened")
	}
	t.Cleanup(func() { os.Remove(lastOpened) })

	if strings.HasPrefix(lastOpened, dir) {
		t.Errorf("output %q should not be in the read-only input directory %q", lastOpened, dir)
	}
	if !strings.HasSuffix(lastOpened, ".md") {
		t.Errorf("temp output file %q should have a .md extension", lastOpened)
	}
	if !strings.Contains(stdout, lastOpened) {
		t.Errorf("stdout %q should mention the output path %q", stdout, lastOpened)
	}
	data, err := os.ReadFile(lastOpened)
	if err != nil {
		t.Fatalf("reading temp output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("temp output file %q should contain the converted subject line", string(data))
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
	// .eml path and writes alongside it, then opens the result.
	input := copyFixture(t, "dragged.eml")

	code, _, stderr := runHelper(t, []string{input}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	want := filepath.Join(filepath.Dir(input), "dragged.md")
	if lastOpened != want {
		t.Fatalf("opened %q, want %q", lastOpened, want)
	}

	data, err := os.ReadFile(lastOpened)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if !strings.Contains(string(data), fixtureSubject) {
		t.Errorf("temp output file %q should contain the converted content", string(data))
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

func TestRun_OpenFailureFallsBackToPrintingPath(t *testing.T) {
	// When the OS opener fails, the path must still be reported so the user can
	// open the file themselves.
	orig := openMarkdownFile
	openMarkdownFile = func(string) error { return errors.New("no opener available") }
	t.Cleanup(func() { openMarkdownFile = orig })

	dir := t.TempDir()
	out := filepath.Join(dir, "result.md")

	code, stdout, stderr := runHelper(t, []string{"--input-file", fixtureEML, "--output-file", out}, "")

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, out) {
		t.Errorf("stdout %q should mention the output path %q so the user can open it", stdout, out)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("output file should still be written when opening fails: %v", err)
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	code, _, _ := runHelper(t, []string{"--nonsense"}, "")
	if code == 0 {
		t.Fatalf("exit code = 0, want non-zero for an unknown flag")
	}
}
