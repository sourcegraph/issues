package search

import "github.com/cockroachdb/errors"

type matchType int

const (
	fileMatch matchType = iota
	repoMatch
	symbolMatch
	commitMatch
)

func (t matchType) MarshalJSON() ([]byte, error) {
	switch t {
	case fileMatch:
		return []byte(`"file"`), nil
	case repoMatch:
		return []byte(`"repo"`), nil
	case symbolMatch:
		return []byte(`"symbol"`), nil
	case commitMatch:
		return []byte(`"commit"`), nil
	default:
		return nil, errors.Errorf("unknown matchType: %d", t)
	}
}
