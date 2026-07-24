package adlists

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zackkitzmiller/tihole/internal/pihole"
)

// gravityChanBuffer sizes the line channel so the streaming producer rarely
// blocks between reader pumps.
const gravityChanBuffer = 64

// gravityLineMsg carries one line pulled from the stream channel, tagged with
// its issuing epoch. ok is false when the channel has been closed (stream end).
type gravityLineMsg struct {
	epoch int
	line  string
	ok    bool
}

// gravityDoneMsg reports that UpdateGravity returned; err is nil on success.
type gravityDoneMsg struct {
	epoch int
	err   error
}

// runGravity returns a fire-and-forget command that blocks in UpdateGravity,
// pushing each progress line into ch and closing ch when the stream ends. It
// returns a gravityDoneMsg carrying the terminal error (nil on success). The
// caller is responsible for cancelling ctx (e.g. on Blur) to stop the stream.
func runGravity(ctx context.Context, api *pihole.Client, ch chan string, epoch int) tea.Cmd {
	return func() tea.Msg {
		err := api.UpdateGravity(ctx, func(line string) {
			ch <- line
		})
		close(ch)
		return gravityDoneMsg{epoch: epoch, err: err}
	}
}

// readGravityLine returns a command that blocks reading a single line from ch
// and reports it as a gravityLineMsg. Re-issued after each received line to
// pump the next; when ch is closed it reports ok=false and is not re-issued.
func readGravityLine(ch chan string, epoch int) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		return gravityLineMsg{epoch: epoch, line: line, ok: ok}
	}
}

// gravitySummary renders a one-line terminal status for the log pane footer.
func gravitySummary(done bool, err error) string {
	switch {
	case err != nil:
		return "gravity failed: " + strings.TrimSpace(err.Error())
	case done:
		return "gravity update complete"
	default:
		return "running gravity update…"
	}
}
