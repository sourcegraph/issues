package query

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/internal/lazyregexp"
)

// SubstituteAliases substitutes field name aliases for their canonical names,
// and substitutes `content:` for pattern nodes.
func SubstituteAliases(searchType SearchType) func(nodes []Node) []Node {
	mapper := func(nodes []Node) []Node {
		return MapParameter(nodes, func(field, value string, negated bool, annotation Annotation) Node {
			if field == "content" {
				if searchType == SearchTypeRegex {
					annotation.Labels.set(Regexp)
				} else {
					annotation.Labels.set(Literal)
				}
				return Pattern{Value: value, Negated: negated, Annotation: annotation}
			}
			field = resolveFieldAlias(field)
			return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
		})
	}
	return mapper
}

// LowercaseFieldNames performs strings.ToLower on every field name.
func LowercaseFieldNames(nodes []Node) []Node {
	return MapParameter(nodes, func(field, value string, negated bool, annotation Annotation) Node {
		return Parameter{Field: strings.ToLower(field), Value: value, Negated: negated, Annotation: annotation}
	})
}

// SubstituteCountAll replaces count:all with count:99999999.
func SubstituteCountAll(nodes []Node) []Node {
	return MapParameter(nodes, func(field, value string, negated bool, annotation Annotation) Node {
		if field == FieldCount && strings.ToLower(value) == "all" {
			return Parameter{Field: field, Value: "99999999", Negated: negated, Annotation: annotation}
		}
		return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
	})
}

var ErrBadGlobPattern = errors.New("syntax error in glob pattern")

// translateCharacterClass translates character classes like [a-zA-Z].
func translateCharacterClass(r []rune, startIx int) (int, string, error) {
	sb := strings.Builder{}
	i := startIx
	lenR := len(r)

	switch r[i] {
	case '!':
		if i < lenR-1 && r[i+1] == ']' {
			// the character class cannot contain just "!"
			return -1, "", ErrBadGlobPattern
		}
		sb.WriteRune('^')
		i++
	case '^':
		sb.WriteString("//^")
		i++
	}

	for i < lenR {
		if r[i] == ']' {
			if i > startIx {
				break
			}
			sb.WriteRune(r[i])
			i++
			continue
		}

		lo := r[i]
		sb.WriteRune(r[i]) // lo
		i++
		if i == lenR {
			// no closing bracket
			return -1, "", ErrBadGlobPattern
		}

		// lo = hi
		if r[i] != '-' {
			continue
		}

		sb.WriteRune(r[i]) // -
		i++
		if i == lenR {
			// no closing bracket
			return -1, "", ErrBadGlobPattern
		}

		if r[i] == ']' {
			continue
		}

		hi := r[i]
		if lo > hi {
			// range is reversed
			return -1, "", ErrBadGlobPattern
		}
		sb.WriteRune(r[i]) // hi
		i++
	}
	if i == lenR {
		return -1, "", ErrBadGlobPattern
	}
	return i - startIx, sb.String(), nil
}

var globSpecialSymbols = map[rune]struct{}{
	'\\': {},
	'*':  {},
	'?':  {},
	'[':  {},
}

// globToRegex converts a glob string to a regular expression.
// We support: *, ?, and character classes [...].
func globToRegex(value string) (string, error) {
	if value == "" {
		return value, nil
	}

	r := []rune(value)
	l := len(r)
	sb := strings.Builder{}

	// Add regex anchor "^" as prefix to all patterns
	sb.WriteRune('^')

	for i := 0; i < l; i++ {
		switch r[i] {
		case '*':
			// **
			if i < l-1 && r[i+1] == '*' {
				sb.WriteString(".*?")
			} else {
				sb.WriteString("[^/]*?")
			}
			// Skip repeated '*'.
			for i < l-1 && r[i+1] == '*' {
				i++
			}
		case '?':
			sb.WriteRune('.')
		case '\\':
			// trailing backslashes are not allowed
			if i == l-1 {
				return "", ErrBadGlobPattern
			}

			sb.WriteRune('\\')
			i++

			// we only support escaping of special characters
			if _, ok := globSpecialSymbols[r[i]]; !ok {
				return "", ErrBadGlobPattern
			}
			sb.WriteRune(r[i])
		case '[':
			if i == l-1 {
				return "", ErrBadGlobPattern
			}
			sb.WriteRune('[')
			i++

			advanced, s, err := translateCharacterClass(r, i)
			if err != nil {
				return "", err
			}

			i += advanced
			sb.WriteString(s)

			sb.WriteRune(']')
		default:
			sb.WriteString(regexp.QuoteMeta(string(r[i])))
		}
	}
	// add regex anchor '$' as suffix to all patterns
	sb.WriteRune('$')
	return sb.String(), nil
}

// globError carries the error message and the name of
// field where the error occurred.
type globError struct {
	field string
	err   error
}

func (g globError) Error() string {
	return g.err.Error()
}

// reporevToRegex is a wrapper around globToRegex that takes care of
// treating repo and rev (as in repo@rev) separately during translation
// from glob to regex.
func reporevToRegex(value string) (string, error) {
	reporev := strings.SplitN(value, "@", 2)
	containsNoRev := len(reporev) == 1
	repo := reporev[0]
	if containsNoRev && ContainsNoGlobSyntax(repo) && !LooksLikeGitHubRepo(repo) {
		repo = fuzzifyGlobPattern(repo)
	}
	repo, err := globToRegex(repo)
	if err != nil {
		return "", err
	}
	value = repo
	if len(reporev) > 1 {
		value = value + "@" + reporev[1]
	}
	return value, nil
}

var globSyntax = lazyregexp.New(`[][*?]`)

func ContainsNoGlobSyntax(value string) bool {
	return !globSyntax.MatchString(value)
}

var gitHubRepoPath = lazyregexp.New(`github\.com\/([a-z\d]+-)*[a-z\d]+\/(.+)`)

// LooksLikeGitHubRepo returns whether string value looks like a valid
// GitHub repo path. This condition is used to guess whether we should
// make a pattern fuzzy, or try it as an exact match.
func LooksLikeGitHubRepo(value string) bool {
	return gitHubRepoPath.MatchString(value)
}

func fuzzifyGlobPattern(value string) string {
	if value == "" {
		return value
	}
	if strings.HasPrefix(value, "github.com") {
		return value + "**"
	}
	return "**" + value + "**"
}

// Globbing translates glob to regexp for fields repo, file, and repohasfile.
func Globbing(nodes []Node) ([]Node, error) {
	var globErrors []globError

	nodes = MapParameter(nodes, func(field, value string, negated bool, annotation Annotation) Node {
		var err error
		switch field {
		case FieldRepo:
			value, err = reporevToRegex(value)
		case FieldFile, FieldRepoHasFile:
			if ContainsNoGlobSyntax(value) {
				value = fuzzifyGlobPattern(value)
			}
			value, err = globToRegex(value)
		}
		if err != nil {
			globErrors = append(globErrors, globError{field: field, err: err})
		}
		return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
	})

	if len(globErrors) == 1 {
		return nil, errors.Errorf("invalid glob syntax in field %s: ", globErrors[0].field)
	}

	if len(globErrors) > 1 {
		fields := globErrors[0].field + ":"

		for _, e := range globErrors[1:] {
			fields += fmt.Sprintf(", %s:", e.field)
		}
		return nil, errors.Errorf("invalid glob syntax in fields %s", fields)
	}

	return nodes, nil
}

func ToNodes(parameters []Parameter) []Node {
	nodes := make([]Node, 0, len(parameters))
	for _, p := range parameters {
		nodes = append(nodes, p)
	}
	return nodes
}

// Converts a flat list of nodes to parameters. Invariant: nodes are parameters.
// This function is intended for internal use only, which assumes the invariant.
func toParameters(nodes []Node) []Parameter {
	var parameters []Parameter
	for _, n := range nodes {
		parameters = append(parameters, n.(Parameter))
	}
	return parameters
}

// Hoist is a heuristic that rewrites simple but possibly ambiguous queries. It
// changes certain expressions in a way that some consider to be more natural.
// For example, the following query without parentheses is interpreted as
// follows in the grammar:
//
// repo:foo a or b and c => (repo:foo a) or ((b) and (c))
//
// This function rewrites the above expression as follows:
//
// repo:foo a or b and c => repo:foo (a or b and c)
//
// Any number of field:value parameters may occur before and after the pattern
// expression separated by or- or and-operators, and these are hoisted out. The
// pattern expression must be contiguous. If not, we want to preserve the
// default interpretation, which corresponds more naturally to groupings with
// field parameters, i.e.,
//
// repo:foo a or b or repo:bar c => (repo:foo a) or (b) or (repo:bar c)
func Hoist(nodes []Node) ([]Node, error) {
	if len(nodes) != 1 {
		return nil, errors.Errorf("heuristic requires one top-level expression")
	}

	expression, ok := nodes[0].(Operator)
	if !ok || expression.Kind == Concat {
		return nil, errors.Errorf("heuristic requires top-level and- or or-expression")
	}

	n := len(expression.Operands)
	var pattern []Node
	var scopeParameters []Parameter
	for i, node := range expression.Operands {
		if i == 0 || i == n-1 {
			scopePart, patternPart, err := PartitionSearchPattern([]Node{node})
			if err != nil || patternPart == nil {
				return nil, errors.New("could not partition first or last expression")
			}
			pattern = append(pattern, patternPart)
			scopeParameters = append(scopeParameters, scopePart...)
			continue
		}
		if !isPatternExpression([]Node{node}) {
			return nil, errors.Errorf("inner expression %s is not a pure pattern expression", node.String())
		}
		pattern = append(pattern, node)
	}
	pattern = MapPattern(pattern, func(value string, negated bool, annotation Annotation) Node {
		annotation.Labels |= HeuristicHoisted
		return Pattern{Value: value, Negated: negated, Annotation: annotation}
	})
	return append(ToNodes(scopeParameters), newOperator(pattern, expression.Kind)...), nil
}

// partition partitions nodes into left and right groups. A node is put in the
// left group if fn evaluates to true, or in the right group if fn evaluates to false.
func partition(nodes []Node, fn func(node Node) bool) (left, right []Node) {
	for _, node := range nodes {
		if fn(node) {
			left = append(left, node)
		} else {
			right = append(right, node)
		}
	}
	return left, right
}

// product appends the list of n elements in right to each of the m rows in
// left. If left is empty, it is initialized with right.
func product(left [][]Node, right []Node) [][]Node {
	result := [][]Node{}
	if len(left) == 0 {
		return append(result, right)
	}

	for _, row := range left {
		newRow := make([]Node, len(row))
		copy(newRow, row)
		result = append(result, append(newRow, right...))
	}
	return result
}

// distribute applies the distributed property to nodes. See the dnf function
// for context. Its first argument takes the current set of prefixes to prepend
// to each term in an or-expression.
func distribute(prefixes [][]Node, nodes []Node) [][]Node {
	for _, node := range nodes {
		switch v := node.(type) {
		case Operator:
			switch v.Kind {
			case Or:
				result := [][]Node{}
				for _, o := range v.Operands {
					var newPrefixes [][]Node
					newPrefixes = distribute(newPrefixes, []Node{o})
					for _, newPrefix := range newPrefixes {
						result = append(result, product(prefixes, newPrefix)...)
					}
				}
				prefixes = result
			case And, Concat:
				prefixes = distribute(prefixes, v.Operands)
			}
		case Parameter, Pattern:
			prefixes = product(prefixes, []Node{v})
		}
	}
	return prefixes
}

// Dnf returns the Disjunctive Normal Form of a query (a flat sequence of
// or-expressions) by applying the distributive property on (possibly nested)
// or-expressions. For example, the query:
//
// (repo:a (file:b OR file:c))
// in DNF becomes:
// (repo:a file:b) OR (repo:a file:c)
//
// Using the DNF expression makes it easy to support general nested queries that
// imply scope, like the one above: We simply evaluate all disjuncts and union
// the results. Note that various optimizations are possible
// during evaluation, but those are separate query pre- or postprocessing steps
// separate from this general transformation.
func Dnf(query []Node) [][]Node {
	return distribute([][]Node{}, query)
}

func substituteOrForRegexp(nodes []Node) []Node {
	isPattern := func(node Node) bool {
		if pattern, ok := node.(Pattern); ok && !pattern.Negated {
			return true
		}
		return false
	}
	newNode := []Node{}
	for _, node := range nodes {
		switch v := node.(type) {
		case Operator:
			if v.Kind == Or {
				patterns, rest := partition(v.Operands, isPattern)
				var values []string
				for _, node := range patterns {
					values = append(values, node.(Pattern).Value)
				}
				valueString := "(" + strings.Join(values, ")|(") + ")"
				newNode = append(newNode, Pattern{Value: valueString})
				if len(rest) > 0 {
					rest = substituteOrForRegexp(rest)
					newNode = newOperator(append(newNode, rest...), Or)
				}
			} else {
				newNode = append(newNode, newOperator(substituteOrForRegexp(v.Operands), v.Kind)...)
			}
		case Parameter, Pattern:
			newNode = append(newNode, node)
		}
	}
	return newNode
}

// fuzzyRegexp interpolates patterns with .*? regular expressions and
// concatenates them. Invariant: len(patterns) > 0.
func fuzzyRegexp(patterns []Pattern) Pattern {
	if len(patterns) == 1 {
		return patterns[0]
	}
	var values []string
	for _, p := range patterns {
		if p.Annotation.Labels.IsSet(Literal) {
			values = append(values, regexp.QuoteMeta(p.Value))
		} else {
			values = append(values, p.Value)
		}
	}
	return Pattern{
		Annotation: Annotation{Labels: Regexp},
		Value:      "(" + strings.Join(values, ").*?(") + ")",
	}
}

// fuzzyRegexp interpolates patterns with spaces and concatenates them.
// Invariant: len(patterns) > 0.
func space(patterns []Pattern) Pattern {
	if len(patterns) == 1 {
		return patterns[0]
	}
	var values []string
	for _, p := range patterns {
		values = append(values, p.Value)
	}

	return Pattern{
		// Preserve labels based on first pattern. Required to
		// distinguish quoted, literal, structural pattern labels.
		Annotation: patterns[0].Annotation,
		Value:      strings.Join(values, " "),
	}
}

// substituteConcat returns a function that concatenates all contiguous patterns
// in the tree, rooted by a concat operator. Concat operators containing negated
// patterns are lifted out: (concat "a" (not "b")) -> ("a" (not "b"))
//
// The callback parameter defines how the function concatenates patterns. The
// return value of callback is substituted in-place in the tree.
func substituteConcat(callback func([]Pattern) Pattern) func(nodes []Node) []Node {
	isPattern := func(node Node) bool {
		if pattern, ok := node.(Pattern); ok && !pattern.Negated {
			return true
		}
		return false
	}

	// define a recursive function to close over callback and isPattern.
	var substituteNodes func(nodes []Node) []Node
	substituteNodes = func(nodes []Node) []Node {
		newNode := []Node{}
		for _, node := range nodes {
			switch v := node.(type) {
			case Parameter, Pattern:
				newNode = append(newNode, node)
			case Operator:
				if v.Kind == Concat {
					// Merge consecutive patterns.
					ps := []Pattern{}
					previous := v.Operands[0]
					if p, ok := previous.(Pattern); ok {
						ps = append(ps, p)
					}
					for _, node := range v.Operands[1:] {
						if isPattern(node) && isPattern(previous) {
							p := node.(Pattern)
							ps = append(ps, p)
							previous = node
							continue
						}
						if len(ps) > 0 {
							newNode = append(newNode, callback(ps))
							ps = []Pattern{}
						}
						newNode = append(newNode, substituteNodes([]Node{node})...)
					}
					if len(ps) > 0 {
						newNode = append(newNode, callback(ps))
					}
				} else {
					newNode = append(newNode, newOperator(substituteNodes(v.Operands), v.Kind)...)
				}
			}
		}
		return newNode
	}
	return substituteNodes
}

// escapeParens is a heuristic used in the context of regular expression search.
// It escapes two kinds of patterns:
//
// 1. Any occurrence of () is converted to \(\).
// In regex () implies the empty string, which is meaningless as a search
// query and probably not what the user intended.
//
// 2. If the pattern ends with a trailing and unescaped (, it is escaped.
// Normally, a pattern like foo.*bar( would be an invalid regexp, and we would
// show no results. But, it is a common and convenient syntax to search for, so
// we convert thsi pattern to interpret a trailing parenthesis literally.
//
// Any other forms are ignored, for example, foo.*(bar is unchanged. In the
// parser pipeline, such unchanged and invalid patterns are rejected by the
// validate function.
func escapeParens(s string) string {
	var i int
	for i := 0; i < len(s); i++ {
		if s[i] == '(' || s[i] == '\\' {
			break
		}
	}

	// No special characters found, so return original string.
	if i >= len(s) {
		return s
	}

	var result []byte
	for i < len(s) {
		switch s[i] {
		case '\\':
			if i+1 < len(s) {
				result = append(result, '\\', s[i+1])
				i += 2 // Next char.
				continue
			}
			i++
			result = append(result, '\\')
		case '(':
			if i+1 == len(s) {
				// Escape a trailing and unescaped ( => \(.
				result = append(result, '\\', '(')
				i++
				continue
			}
			if i+1 < len(s) && s[i+1] == ')' {
				// Escape () => \(\).
				result = append(result, '\\', '(', '\\', ')')
				i += 2 // Next char.
				continue
			}
			result = append(result, s[i])
			i++
		default:
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}

// escapeParensHeuristic escapes certain parentheses in search patterns (see escapeParens).
func escapeParensHeuristic(nodes []Node) []Node {
	return MapPattern(nodes, func(value string, negated bool, annotation Annotation) Node {
		if !annotation.Labels.IsSet(Quoted) {
			value = escapeParens(value)
		}
		return Pattern{
			Value:      value,
			Negated:    negated,
			Annotation: annotation,
		}
	})
}

// Map pipes query through one or more query transformer functions.
func Map(query []Node, fns ...func([]Node) []Node) []Node {
	for _, fn := range fns {
		query = fn(query)
	}
	return query
}

func FuzzifyRegexPatterns(nodes []Node) []Node {
	return MapParameter(nodes, func(field string, value string, negated bool, annotation Annotation) Node {
		if field == FieldRepo || field == FieldFile || field == FieldRepoHasFile {
			value = strings.TrimSuffix(value, "$")
		}
		return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
	})
}

// concatRevFilters removes rev: filters from parameters and attaches their value as @rev to the repo: filters.
// Invariant: Guaranteed to succeed on a validat Basic query.
func ConcatRevFilters(b Basic) Basic {
	var revision string
	nodes := MapField(ToNodes(b.Parameters), FieldRev, func(value string, _ bool) Node {
		revision = value
		return nil // remove this node
	})
	if revision == "" {
		return b
	}
	modified := MapField(nodes, FieldRepo, func(value string, negated bool) Node {
		if !negated {
			return Parameter{Value: value + "@" + revision, Field: FieldRepo, Negated: negated}
		}
		return Parameter{Value: value, Field: FieldRepo, Negated: negated}
	})
	return Basic{Parameters: toParameters(modified), Pattern: b.Pattern}
}

// labelStructural converts Literal labels to Structural labels. Structural
// queries are parsed the same as literal queries, we just convert the labels as
// a postprocessing step to keep the parser lean.
func labelStructural(nodes []Node) []Node {
	return MapPattern(nodes, func(value string, negated bool, annotation Annotation) Node {
		annotation.Labels.unset(Literal)
		annotation.Labels.set(Structural)
		return Pattern{
			Value:      value,
			Negated:    negated,
			Annotation: annotation,
		}
	})
}

// ellipsesForHoles substitutes ellipses ... for :[_] holes in structural search queries.
func ellipsesForHoles(nodes []Node) []Node {
	return MapPattern(nodes, func(value string, negated bool, annotation Annotation) Node {
		return Pattern{
			Value:      strings.ReplaceAll(value, "...", ":[_]"),
			Negated:    negated,
			Annotation: annotation,
		}
	})
}

func OverrideField(nodes []Node, field, value string) []Node {
	// First remove any fields that exist.
	nodes = MapField(nodes, field, func(_ string, _ bool) Node {
		return nil
	})
	return newOperator(append(nodes, Parameter{Field: field, Value: value}), And)
}

// OmitField removes all fields `field` from a query. The `field` string
// should be the canonical name and not an alias ("repo", not "r").
func OmitField(q Q, field string) string {
	return StringHuman(MapField(q, field, func(_ string, _ bool) Node {
		return nil
	}))
}

// addRegexpField adds a new expr to the query with the given field and pattern
// value. The nonnegated field is assumed to associate with a regexp value. The
// pattern value is assumed to be unquoted.
//
// It tries to remove redundancy in the result. For example, given
// a query like "x:foo", if given a field "x" with pattern "foobar" to add,
// it will return a query "x:foobar" instead of "x:foo x:foobar". It is not
// guaranteed to always return the simplest query.
func AddRegexpField(q Q, field, pattern string) string {
	var modified bool
	q = MapParameter(q, func(gotField, value string, negated bool, annotation Annotation) Node {
		if field == gotField && strings.Contains(pattern, value) {
			value = pattern
			modified = true
		}
		return Parameter{
			Field:      gotField,
			Value:      value,
			Negated:    negated,
			Annotation: annotation,
		}
	})

	if !modified {
		// use newOperator to reduce And nodes when adding a parameter to the query toplevel.
		q = newOperator(append(q, Parameter{Field: field, Value: pattern}), And)
	}
	return StringHuman(q)
}

func identity(nodes []Node) ([]Node, error) {
	return nodes, nil
}

// Converts a parse tree to a basic query by attempting to obtain a valid partition.
func ToBasicQuery(nodes []Node) (Basic, error) {
	parameters, pattern, err := PartitionSearchPattern(nodes)
	if err != nil {
		return Basic{}, err
	}
	return Basic{Parameters: parameters, Pattern: pattern}, nil
}

// Identity is the identity transformer for basic queries.
func Identity(b Basic) Basic {
	return b
}

// PatternToFile transforms a search query such that `file:` is prefixed to the
// pattern. This transformation is used for generating suggestions. Succeeds
// only when the pattern expression is an atom.
func PatternToFile(b Basic) Basic {
	if p, ok := b.Pattern.(Pattern); ok && !p.Negated {
		b.Parameters = append(b.Parameters, Parameter{Field: "file", Value: p.Value})
		return b
	}
	return b
}

func remove(s []Parameter, i int) []Parameter {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

// typeRepoToFilter transforms a search query of the form `type:repo <pattern>`
// to repo:<pattern>. It is a normalization to handle repo searches consistently
// in the backend.
func typeRepoToFilter(b Basic) Basic {
	seen := false
	for i, p := range b.Parameters {
		if p.Field == FieldType {
			seen = true
			remove(b.Parameters, i)
		}
	}

	if !seen {
		return b
	}

	if IsPatternAtom(b) {
		p := b.Pattern.(Pattern)
		ann := p.Annotation
		value := p.Value
		if ann.Labels.IsSet(Quoted) || ann.Labels.IsSet(Literal) || ann.Labels.IsSet(Structural) {
			// Respect search type: If the pattern is literal, then
			// interpet the value literally for the repo filter.
			value = regexp.QuoteMeta(p.Value)
		}
		return b.MapPattern(nil).MapParameters(append(b.Parameters, Parameter{
			Field:   FieldRepo,
			Value:   value,
			Negated: false,
		}))
	}
	return b // unhandled case
}
