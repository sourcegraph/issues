// Validation logic for TextPatternInfo
package search

import (
	"regexp/syntax"
)

func (p *TextPatternInfo) IsEmpty() bool {
	return p.Pattern == "" && p.ExcludePattern == "" && len(p.IncludePatterns) == 0
}

func (p *TextPatternInfo) Validate() error {
	if p.IsRegExp {
		if _, err := syntax.Parse(p.Pattern, syntax.Perl); err != nil {
			return err
		}
	}

	if p.ExcludePattern != "" {
		if _, err := syntax.Parse(p.ExcludePattern, syntax.Perl); err != nil {
			return err
		}
	}
	for _, expr := range p.IncludePatterns {
		if _, err := syntax.Parse(expr, syntax.Perl); err != nil {
			return err
		}
	}

	return nil
}
