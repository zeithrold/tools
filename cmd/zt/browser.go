package main

import (
	"os/exec"
	"runtime"
)

// openInBrowser launches the platform URL handler without a shell. The caller
// validates that the URL is a trusted HTTPS device-verification page first.
func openInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
