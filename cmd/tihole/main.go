// Command tihole is a terminal UI for managing PiHole v6 instances.
package main

import (
	"fmt"
	"os"

	"github.com/zackkitzmiller/tihole/internal/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tihole:", err)
		os.Exit(1)
	}
}
