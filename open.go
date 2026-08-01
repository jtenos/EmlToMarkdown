package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// openMarkdownFile opens the file at filePath in the host OS's default handler
// for its extension. It is a package-level variable so tests can substitute a
// stub in place of launching a real application.
var openMarkdownFile = OpenMarkdownFile

// OpenMarkdownFile opens a file using the host OS's default application.
func OpenMarkdownFile(filePath string) error {
	var cmd *exec.Cmd

	switch {
	case isWSL():
		// WSL: use wslview (part of wslu) when available; it translates the path
		// to Windows form and hands it to the default Windows app. Otherwise fall
		// back to cmd.exe's start.
		if _, err := exec.LookPath("wslview"); err == nil {
			cmd = exec.Command("wslview", filePath)
		} else {
			cmd = exec.Command("cmd.exe", "/c", "start", "", filePath)
		}

	case runtime.GOOS == "windows":
		// Windows: "start" launches the default handler for the file extension.
		cmd = exec.Command("cmd.exe", "/c", "start", "", filePath)

	case runtime.GOOS == "darwin":
		// macOS: "open" launches the default application for the file.
		cmd = exec.Command("open", filePath)

	case runtime.GOOS == "linux":
		// Native Linux: xdg-open opens files using the desktop environment's
		// default app.
		cmd = exec.Command("xdg-open", filePath)

	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// isWSL reports whether the process is running inside Windows Subsystem for Linux.
func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	// WSL kernels include "microsoft" (or "wsl") in the kernel release string.
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}
