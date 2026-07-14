package components

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zackkitzmiller/tihole/internal/theme"
)

// paletteMaxRows caps the visible command list height.
const paletteMaxRows = 8

// Command is one entry in the command palette. Run is the tea.Cmd executed when
// the command is chosen; it typically emits a core message (NavigateMsg,
// SetThemeMsg, …) the root already knows how to handle.
type Command struct {
	Title string
	Desc  string
	Run   tea.Cmd
}

// Palette is a Ctrl-K fuzzy command launcher rendered as a centered overlay.
// The root owns it, feeds it a fresh command set on open, and forwards key
// messages while it is active.
type Palette struct {
	input    textinput.Model
	all      []Command
	filtered []Command
	cursor   int
	active   bool
}

// NewPalette constructs an inactive palette with its search input.
func NewPalette() Palette {
	in := textinput.New()
	in.Prompt = "› "
	in.Placeholder = "type a command…"
	in.CharLimit = 64
	return Palette{input: in}
}

// Active reports whether the palette is currently shown.
func (p Palette) Active() bool { return p.active }

// Open activates the palette with the given command set and focuses the input.
func (p Palette) Open(cmds []Command) (Palette, tea.Cmd) {
	p.all = cmds
	p.filtered = cmds
	p.cursor = 0
	p.active = true
	p.input.SetValue("")
	return p, p.input.Focus()
}

// Close deactivates the palette and blurs the input.
func (p Palette) Close() Palette {
	p.active = false
	p.input.Blur()
	return p
}

// Update handles navigation, filtering, and selection while active. It returns
// the (possibly closed) palette and any command produced: the selected
// Command.Run on enter, or the input's own cursor-blink cmd otherwise.
func (p Palette) Update(msg tea.Msg) (Palette, tea.Cmd) {
	if !p.active {
		return p, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return p, cmd
	}

	switch key.String() {
	case "esc":
		return p.Close(), nil
	case "up", "ctrl+p":
		if p.cursor > 0 {
			p.cursor--
		}
		return p, nil
	case "down", "ctrl+n":
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
		return p, nil
	case "enter":
		if p.cursor >= 0 && p.cursor < len(p.filtered) {
			run := p.filtered[p.cursor].Run
			return p.Close(), run
		}
		return p.Close(), nil
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.filtered = filterCommands(p.all, p.input.Value())
	if p.cursor >= len(p.filtered) {
		p.cursor = maxInt0(len(p.filtered)-1, 0)
	}
	return p, cmd
}

// filterCommands returns commands whose title or description contains every
// whitespace-separated term of the query (case-insensitive). Empty query
// returns all commands unchanged.
func filterCommands(cmds []Command, query string) []Command {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return cmds
	}
	terms := strings.Fields(q)
	var out []Command
	for _, c := range cmds {
		hay := strings.ToLower(c.Title + " " + c.Desc)
		if containsAll(hay, terms) {
			out = append(out, c)
		}
	}
	return out
}

func containsAll(hay string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

// Render draws the palette overlay centered in the given area. Returns "" when
// inactive.
func (p Palette) Render(th *theme.Theme, width, height int) string {
	if !p.active {
		return ""
	}

	boxWidth := clampInt(width-8, 30, 70)

	title := lipgloss.NewStyle().
		Foreground(th.Accent).
		Bold(true).
		Render("Command palette")

	input := lipgloss.NewStyle().
		Foreground(th.Text).
		Width(boxWidth - 2).
		Render(p.input.View())

	var rows []string
	rows = append(rows, title, "", input, "")

	if len(p.filtered) == 0 {
		rows = append(rows, th.SubtleStyle().Render("no matching commands"))
	}

	start, end := windowBounds(p.cursor, len(p.filtered), paletteMaxRows)
	for i := start; i < end; i++ {
		rows = append(
			rows,
			p.renderRow(th, p.filtered[i], i == p.cursor, boxWidth-2),
		)
	}

	rows = append(
		rows,
		"",
		th.SubtleStyle().Render("↑↓ move · enter run · esc close"),
	)

	box := lipgloss.NewStyle().
		Background(th.Panel).
		Foreground(th.Text).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Accent).
		Padding(1, 2).
		Width(boxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (p Palette) renderRow(
	th *theme.Theme,
	c Command,
	selected bool,
	width int,
) string {
	title := c.Title
	desc := c.Desc
	if selected {
		row := lipgloss.NewStyle().
			Foreground(th.Surface).
			Background(th.Accent).
			Bold(true).
			Width(width).
			Padding(0, 1).
			Render(title)
		if desc != "" {
			row += "\n" + lipgloss.NewStyle().
				Foreground(th.Subtle).
				Padding(0, 1).
				Render(desc)
		}
		return row
	}
	row := lipgloss.NewStyle().Foreground(th.Text).Padding(0, 1).Render(title)
	if desc != "" {
		row += "  " + th.SubtleStyle().Render(desc)
	}
	return row
}

// windowBounds returns a [start,end) window of size at most max that keeps the
// cursor visible.
func windowBounds(cursor, n, max int) (int, int) {
	if n <= max {
		return 0, n
	}
	start := cursor - max/2
	if start < 0 {
		start = 0
	}
	end := start + max
	if end > n {
		end = n
		start = end - max
	}
	return start, end
}

func maxInt0(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
