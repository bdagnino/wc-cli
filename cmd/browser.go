package cmd

import (
	"os/exec"
	"runtime"
)

// openInBrowser opens url in the user's default browser. It shells out to the
// platform's standard handler rather than pulling in a dependency, and starts
// the process without waiting so the CLI returns immediately.
func openInBrowser(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name = "open"
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default: // linux, *bsd, etc.
		name = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(name, args...).Start()
}
