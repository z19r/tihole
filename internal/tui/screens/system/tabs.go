package system

// tab identifies one of the System/Tools sub-screens. Tabs are switched by
// tab/f/←/→ and each loads its own data lazily (epoch-guarded) when entered.
type tab int

const (
	tabInfo tab = iota
	tabMessages
	tabNetwork
	tabLog
	tabActions
	tabCount // sentinel: number of tabs
)

// tabLabels is the header chip text for each tab, in render order.
var tabLabels = map[tab]string{
	tabInfo:     "Info",
	tabMessages: "Messages",
	tabNetwork:  "Network",
	tabLog:      "Log tail",
	tabActions:  "Actions",
}

// label returns the human-readable tab label.
func (t tab) label() string {
	if l, ok := tabLabels[t]; ok {
		return l
	}
	return "Info"
}

// next returns the following tab, wrapping around.
func (t tab) next() tab { return (t + 1) % tabCount }

// prev returns the preceding tab, wrapping around.
func (t tab) prev() tab { return (t + tabCount - 1) % tabCount }
