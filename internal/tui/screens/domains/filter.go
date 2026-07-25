package domains

import "github.com/zackkitzmiller/tihole/internal/pihole"

// filterTab selects which subset of the fetched domains is shown. The tabs
// cycle
// through the four allow/deny × exact/regex combinations plus an "all" view.
type filterTab int

const (
	filterAllowExact filterTab = iota
	filterDenyExact
	filterAllowRegex
	filterDenyRegex
	filterAll
	filterCount // sentinel: number of tabs
)

// filterLabels is the chip text shown for each tab.
var filterLabels = map[filterTab]string{
	filterAllowExact: "allow · exact",
	filterDenyExact:  "deny · exact",
	filterAllowRegex: "allow · regex",
	filterDenyRegex:  "deny · regex",
	filterAll:        "all",
}

// label returns the human-readable chip label for the tab.
func (f filterTab) label() string {
	if l, ok := filterLabels[f]; ok {
		return l
	}
	return "all"
}

// next returns the following tab, wrapping around.
func (f filterTab) next() filterTab {
	return (f + 1) % filterCount
}

// prev returns the preceding tab, wrapping around.
func (f filterTab) prev() filterTab {
	return (f + filterCount - 1) % filterCount
}

// typeKind resolves the tab's domain type and kind. Only meaningful for the
// four
// concrete tabs; filterAll is handled separately by filterDomains.
func (f filterTab) typeKind() (pihole.DomainType, pihole.DomainKind) {
	switch f {
	case filterAllowExact:
		return pihole.DomainAllow, pihole.KindExact
	case filterDenyExact:
		return pihole.DomainDeny, pihole.KindExact
	case filterAllowRegex:
		return pihole.DomainAllow, pihole.KindRegex
	case filterDenyRegex:
		return pihole.DomainDeny, pihole.KindRegex
	default:
		return pihole.DomainAllow, pihole.KindExact
	}
}

// filterDomains returns the subset of domains matching the given tab. The
// result
// is a fresh slice, never a view onto the input.
func filterDomains(domains []pihole.Domain, f filterTab) []pihole.Domain {
	if f == filterAll {
		out := make([]pihole.Domain, len(domains))
		copy(out, domains)
		return out
	}

	wantType, wantKind := f.typeKind()
	out := make([]pihole.Domain, 0, len(domains))
	for _, d := range domains {
		if d.Type == string(wantType) && d.Kind == string(wantKind) {
			out = append(out, d)
		}
	}
	return out
}
