// Package browser opens URLs in the user's default web browser.
package browser

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open starts the default browser for rawURL.
func Open(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}
	if configured := os.Getenv("BROWSER"); configured != "" {
		parts := strings.Fields(configured)
		cmd := exec.Command(parts[0], append(parts[1:], rawURL)...)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Available reports whether a browser launcher looks available.
func Available() bool {
	if os.Getenv("BROWSER") != "" {
		return true
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return true
	}
	_, err := exec.LookPath("xdg-open")
	return err == nil
}
