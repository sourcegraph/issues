package streaming

import (
	"container/heap"
	"sort"
	"strings"
)

type Filter struct {
	Value string

	// Label is the string to be displayed in the UI.
	Label string

	// Count is the number of matches in a particular repository. Only used
	// for `repo:` filters.
	Count int

	// IsLimitHit is true if the results returned for a repository are
	// incomplete.
	IsLimitHit bool

	// Kind of filter. Should be "repo", "file", or "lang".
	Kind string

	// important is used to prioritize the order that filters appear in.
	important bool
}

// Less returns true if f is more important the o.
func (f *Filter) Less(o *Filter) bool {
	if f.important != o.important {
		// Prefer more important
		return f.important
	}
	if f.Count != o.Count {
		// Prefer higher count
		return f.Count > o.Count
	}
	// Order alphabetically for equal scores.
	return strings.Compare(f.Value, o.Value) < 0

}

// filters is a map of filter values to the Filter.
type filters map[string]*Filter

// Add the count to the filter with value.
func (m filters) Add(value string, label string, count int32, limitHit bool, kind string) {
	sf, ok := m[value]
	if !ok {
		sf = &Filter{
			Value:      value,
			Label:      label,
			Count:      int(count),
			IsLimitHit: limitHit,
			Kind:       kind,
		}
		m[value] = sf
	} else {
		sf.Count += int(count)
	}
}

// MarkImportant sets the filter with value as important. Can only be called
// after Add.
func (m filters) MarkImportant(value string) {
	m[value].important = true
}

// computeOpts are the options for calling filters.Compute.
type computeOpts struct {
	// MaxRepos is the maximum number of filters to return with kind repo.
	MaxRepos int

	// MaxOther is the maximum number of filters to return which are not repo.
	MaxOther int
}

// Compute returns an ordered slice of Filter to present to the user.
func (m filters) Compute(opts computeOpts) []*Filter {
	repos := filterHeap{max: opts.MaxRepos}
	other := filterHeap{max: opts.MaxOther}
	for _, f := range m {
		if f.Kind == "repo" {
			repos.Add(f)
		} else {
			other.Add(f)
		}
	}

	all := append(repos.filterSlice, other.filterSlice...)
	sort.Sort(all)

	return all
}

type filterSlice []*Filter

func (fs filterSlice) Len() int {
	return len(fs)
}

func (fs filterSlice) Less(i, j int) bool {
	return fs[i].Less(fs[j])
}

func (fs filterSlice) Swap(i, j int) {
	fs[i], fs[j] = fs[j], fs[i]
}

// filterHeap allows us to avoid creating an O(N) slice, sorting it O(NlogN)
// and then keeping the max elements. Instead we use a heap to use O(max)
// space and O(Nlogmax) runtime.
type filterHeap struct {
	filterSlice
	max int
}

func (h *filterHeap) Add(f *Filter) {
	if len(h.filterSlice) < h.max {
		// Less than max, we keep the filter.
		heap.Push(h, f)
	} else if h.max > 0 && f.Less(h.filterSlice[0]) {
		// f is more important than the least important filter we have
		// kept. So Pop that filter away and add in f. We should keep the
		// invariant that len == h.max.
		heap.Pop(h)
		heap.Push(h, f)
	}
}

func (h *filterHeap) Less(i, j int) bool {
	// We want a max heap so that the head of the heap is the least important
	// value we have kept so far.
	return h.filterSlice[j].Less(h.filterSlice[i])
}

func (h *filterHeap) Push(x interface{}) {
	h.filterSlice = append(h.filterSlice, x.(*Filter))
}

func (h *filterHeap) Pop() interface{} {
	old := h.filterSlice
	n := len(old)
	x := old[n-1]
	h.filterSlice = old[0 : n-1]
	return x
}
