// Validation logic for TextPatternInfo
package search

func (p *TextPatternInfo) IsEmpty() bool {
	return p.Pattern == "" && p.ExcludePattern == "" && len(p.IncludePatterns) == 0
}
