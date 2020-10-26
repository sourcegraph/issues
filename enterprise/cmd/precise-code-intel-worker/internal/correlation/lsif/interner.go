package lsif

import (
	"strconv"
	"sync"
)

// Interner converts strings into unique identifers. Submitting the same byte value to
// the interner will result in the same identifier being produced. Each unique input is
// guaranteed to have a unique output (no two inputs share the same identifier). The
// identifier space of two distinct interner instances may overlap.
//
// Assumption: The output of LSIF indexers will not generally mix types of identifiers.
// If integers are used, they are used for all ids. If strings are used, they are used
// for all ids.
type Interner struct {
	sync.RWMutex
	m map[string]int
}

// NewInterner creates a new empty interner.
func NewInterner() *Interner {
	return &Interner{
		m: map[string]int{},
	}
}

// Intern returns the unique identifier for the given byte value. The byte value should
// be a raw LSIF input identifier, which should be a JSON-encoded number or quoted string.
// This method is safe to call from multiple goroutines.
func (i *Interner) Intern(raw []byte) (int, error) {
	if len(raw) == 0 {
		// No identifier supplied
		return 0, nil
	}

	if raw[0] != '"' {
		// Not a string, expect a number
		return strconv.Atoi(string(raw))
	}

	// Generate a numeric identifier for the de-quoted string
	s := string(raw[1 : len(raw)-1])

	// See if this is an "inty" string (e.g., "1234"). We can use a
	// fast-path here that does not need to lock or stash the string
	// value in a map.
	if v, err := strconv.Atoi(s); err == nil {
		return v, nil
	}

	i.RLock()
	v, ok := i.m[s]
	i.RUnlock()
	if ok {
		return v, nil
	}

	i.Lock()
	defer i.Unlock()

	v, ok = i.m[s]
	if !ok {
		// Generate and stash a new identifier
		v = len(i.m) + 1
		i.m[s] = v
	}

	return v, nil
}
