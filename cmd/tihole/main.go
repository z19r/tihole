// Command tihole is a terminal UI for managing PiHole v6 instances.
package main

import (
	"fmt"
	"os"

	"github.com/zackkitzmiller/tihole/internal/tui"
)

const usage = `tihole — a terminal UI for PiHole v6

Usage:
  tihole           Launch the dashboard (default)
  tihole config    Launch the config editor to add or fix an instance
  tihole help      Show this help

Config lives at ~/.config/tihole/config.yaml. A wrong address or password no
longer aborts startup: errors are shown in-app, and a broken instance opens the
config editor automatically.`

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
	case "help", "-h", "--help":
		fmt.Println(usage)
		return nil
	case "":
		return tui.Run()
	default:
		return fmt.Errorf("unknown command %q (try `tihole help`)", cmd)
	}
}
