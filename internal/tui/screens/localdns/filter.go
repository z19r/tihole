package localdns

// recordKind selects which local-DNS record family is shown: A/AAAA host
// records or CNAME records. The header chip is toggled with f/←/→.
type recordKind int

const (
	kindHosts recordKind = iota
	kindCNAMEs
	kindCount // sentinel: number of kinds
)

// kindLabels is the chip text shown for each kind.
var kindLabels = map[recordKind]string{
	kindHosts:  "Hosts (A/AAAA)",
	kindCNAMEs: "CNAMEs",
}

// label returns the human-readable chip label for the kind.
func (k recordKind) label() string {
	if l, ok := kindLabels[k]; ok {
		return l
	}
	return "Hosts (A/AAAA)"
}

// next returns the following kind, wrapping around.
func (k recordKind) next() recordKind {
	return (k + 1) % kindCount
}

// prev returns the preceding kind, wrapping around.
func (k recordKind) prev() recordKind {
	return (k + kindCount - 1) % kindCount
}
