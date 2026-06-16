// Command wcup follows the 2026 FIFA World Cup from the terminal.
package main

import (
	"runtime/debug"

	"github.com/bdagnino/wc-cli/cmd"
)

// version is set at build time via -ldflags "-X main.version=..." for release
// binaries. For `go install`ed builds it falls back to the embedded module
// version (see resolveVersion).
var version = "dev"

func main() {
	cmd.Execute(resolveVersion())
}

// resolveVersion prefers the linker-injected version, then the module version
// recorded by `go install module@vX.Y.Z`, and finally "dev".
func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return version
}
