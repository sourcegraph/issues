package query

import "sort"

// Labels are general-purpose annotations that store information about a node.
type labels uint8

const (
	None    labels = 0
	Literal        = 1 << iota
	Regexp
	Quoted
	HeuristicParensAsPatterns
	HeuristicDanglingParens
	HeuristicHoisted
	Structural
)

var allLabels = map[labels]string{
	None:                      "None",
	Literal:                   "Literal",
	Regexp:                    "Regexp",
	Quoted:                    "Quoted",
	HeuristicParensAsPatterns: "HeuristicParensAsPatterns",
	HeuristicDanglingParens:   "HeuristicDanglingParens",
	HeuristicHoisted:          "HeuristicHoisted",
	Structural:                "Structural",
}

func (l *labels) isSet(label labels) bool {
	return *l&label != 0
}

func (l *labels) set(label labels) {
	*l |= label
}

func (l *labels) unset(label labels) {
	*l &^= label
}

func (l *labels) String() []string {
	if *l == 0 {
		return []string{"None"}
	}
	var s []string
	for k, v := range allLabels {
		if l.isSet(k) {
			s = append(s, v)
		}
	}
	sort.Strings(s)
	return s
}
