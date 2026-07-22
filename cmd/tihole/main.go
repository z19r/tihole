// Command tihole is a terminal UI for managing PiHole v6 instances.
package main

import (
	"fmt"
	"os"

	"github.com/z19r/tihole/internal/tui"
)

const usage = `tihole — a terminal UI for PiHole v6

Usage:
  tihole                 Launch the dashboard (default)
  tihole config          Launch the config editor to add or fix an instance
  tihole changelog-sync  Regenerate site/src/changelog.js from CHANGELOG.md
  tihole version         Print the tihole version
  tihole help            Show this help

Config lives at ~/.config/tihole/config.yaml. A wrong address or password no
longer aborts startup: errors are shown in-app, and a broken instance opens the
config editor automatically.`

// version is stamped at build time via -ldflags "-X main.version=...".
// It defaults to "dev" for local builds.
var version = "dev"

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "tihole:", err)
		os.Exit(1)
	}
}

// dispatch routes the first argument to a subcommand, defaulting to the TUI.
func dispatch(args []string) error {
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "config", "-c", "--config":
		return tui.RunConfig()
	case "changelog-sync":
		return runChangelogSync()
	case "help", "-h", "--help":
		fmt.Println(usage)
		return nil
	case "version", "-v", "--version":
		fmt.Printf("tihole %s\n", version)
		return nil
	case "":
		return tui.Run()
	default:
		return fmt.Errorf("unknown command %q (try `tihole help`)", cmd)
	}
}
