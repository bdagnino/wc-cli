// Command wcup follows the 2026 FIFA World Cup from the terminal.
package main

import "github.com/bdagnino/wc-cli/cmd"

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	cmd.Execute(version)
}
