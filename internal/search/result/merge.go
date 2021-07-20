package result

// Union performs a merge of results, merging line matches when they occur in
// the same file.
func Union(left, right []Match) []Match {
	dedup := NewDeduper()
	// Add results to maps for deduping
	for _, result := range left {
		dedup.Add(result)
	}
	for _, result := range right {
		dedup.Add(result)
	}
	return dedup.Results()
}

// Intersect performs a merge of match results, merging line matches for files
// contained in both result sets.
func Intersect(left, right []Match) []Match {
	rightMap := make(map[Key]Match, len(right))
	for _, r := range right {
		rightMap[r.Key()] = r
	}

	merged := left[:0]
	for _, l := range left {
		r := rightMap[l.Key()]
		if r == nil {
			continue
		}
		if leftFileMatch, ok := l.(*FileMatch); ok {
			leftFileMatch.AppendMatches(r.(*FileMatch)) // key matches, so we know it's a file match
		}
		merged = append(merged, l)
	}
	return merged
}
