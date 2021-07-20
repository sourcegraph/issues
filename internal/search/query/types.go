package query

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type ExpectedOperand struct {
	Msg string
}

func (e *ExpectedOperand) Error() string {
	return e.Msg
}

type UnsupportedError struct {
	Msg string
}

func (e *UnsupportedError) Error() string {
	return e.Msg
}

type SearchType int

const (
	SearchTypeRegex SearchType = iota
	SearchTypeLiteral
	SearchTypeStructural
)

func (s SearchType) String() string {
	switch s {
	case SearchTypeRegex:
		return "regex"
	case SearchTypeLiteral:
		return "literal"
	case SearchTypeStructural:
		return "structural"
	default:
		return fmt.Sprintf("unknown{%d}", s)
	}
}

// QueryInfo is an interface for accessing query values that drive our search logic.
// It will be removed in favor of a cleaner query API to access values.
type QueryInfo interface {
	Count() *int
	Archived() *YesNoOnly
	Fork() *YesNoOnly
	Timeout() *time.Duration
	Repositories() (repos []string, negated []string)
	RegexpPatterns(field string) (values, negatedValues []string)
	StringValues(field string) (values, negatedValues []string)
	StringValue(field string) (value, negatedValue string)
	Values(field string) []*Value
	Fields() map[string][]*Value
	BoolValue(field string) bool
	IsCaseSensitive() bool
}

// A query plan represents a set of disjoint queries for the search engine to
// execute. The result of executing a plan is the union of individual query results.
type Plan []Basic

// ToParseTree models a plan as a parse tree of an Or-expression on plan queries.
func (p Plan) ToParseTree() Q {
	nodes := make([]Node, 0, len(p))
	for _, basic := range p {
		operands := basic.ToParseTree()
		nodes = append(nodes, newOperator(operands, And)...)
	}
	return Q(newOperator(nodes, Or))
}

// Basic represents a leaf expression to evaluate in our search engine. A basic
// query comprises:
//   (1) a single search pattern expression, which may contain
//       'and' or 'or' operators; and
//   (2) parameters that scope the evaluation of search
//       patterns (e.g., to repos, files, etc.).
type Basic struct {
	Pattern    Node
	Parameters []Parameter
}

func (b Basic) ToParseTree() Q {
	var nodes []Node
	for _, n := range b.Parameters {
		nodes = append(nodes, Node(n))
	}
	if b.Pattern == nil {
		return nodes
	}
	nodes = append(nodes, b.Pattern)
	if hoisted, err := Hoist(nodes); err == nil {
		return hoisted
	}
	return nodes
}

// MapPattern returns a copy of a basic query with updated pattern.
func (b Basic) MapPattern(pattern Node) Basic {
	return Basic{Parameters: b.Parameters, Pattern: pattern}
}

// MapParameters returns a copy of a basic query with updated parameters.
func (b Basic) MapParameters(parameters []Parameter) Basic {
	return Basic{Parameters: parameters, Pattern: b.Pattern}
}

// AddCount adds a count parameter to a basic query. Behavior of AddCount on a
// query that already has a count parameter is undefined.
func (b Basic) AddCount(count int) Basic {
	return b.MapParameters(append(b.Parameters, Parameter{
		Field: "count",
		Value: strconv.FormatInt(int64(count), 10),
	}))
}

// GetCount returns the string value of the "count:" field. Returns empty string if none.
func (b Basic) GetCount() string {
	var countStr string
	VisitField(ToNodes(b.Parameters), "count", func(value string, _ bool, _ Annotation) {
		countStr = value
	})
	return countStr
}

// GetTimeout returns the time.Duration value from the `timeout:` field.
func (b Basic) GetTimeout() *time.Duration {
	var timeout *time.Duration
	VisitField(ToNodes(b.Parameters), FieldTimeout, func(value string, _ bool, _ Annotation) {
		t, err := time.ParseDuration(value)
		if err != nil {
			panic(fmt.Sprintf("Value %q for timeout cannot be parsed as an duration: %s", value, err))
		}
		timeout = &t
	})
	return timeout
}

// MapCount returns a copy of a basic query with a count parameter set.
func (b Basic) MapCount(count int) Basic {
	parameters := MapParameter(ToNodes(b.Parameters), func(field, value string, negated bool, annotation Annotation) Node {
		if field == "count" {
			value = strconv.FormatInt(int64(count), 10)
		}
		return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
	})
	return Basic{Parameters: toParameters(parameters), Pattern: b.Pattern}
}

func (b Basic) String() string {
	return fmt.Sprintf("%s %s", Q(ToNodes(b.Parameters)).String(), Q([]Node{b.Pattern}).String())
}

func (b Basic) VisitParameter(field string, f func(value string, negated bool, annotation Annotation)) {
	for _, p := range b.Parameters {
		if p.Field == field {
			f(p.Value, p.Negated, p.Annotation)
		}
	}
}

// HasPatternLabel returns whether a pattern atom has a specified label.
func (b Basic) HasPatternLabel(label labels) bool {
	if b.Pattern == nil {
		return false
	}
	if _, ok := b.Pattern.(Pattern); !ok {
		// Basic query is not atomic.
		return false
	}
	annot := b.Pattern.(Pattern).Annotation
	return annot.Labels.IsSet(label)
}

func (b Basic) IsLiteral() bool {
	return b.HasPatternLabel(Literal)
}

func (b Basic) IsRegexp() bool {
	return b.HasPatternLabel(Regexp)
}

func (b Basic) IsStructural() bool {
	return b.HasPatternLabel(Structural)
}

// FindParameter calls f on parameters matching field in b.
func (b Basic) FindParameter(field string, f func(value string, negated bool, annotation Annotation)) {
	for _, p := range b.Parameters {
		if p.Field == field {
			f(p.Value, p.Negated, p.Annotation)
			break
		}
	}
}

// FindValue returns the first value of a parameter matching field in b. It
// doesn't inspect whether the field is negated.
func (b Basic) FindValue(field string) (value string) {
	var found string
	b.FindParameter(field, func(v string, _ bool, _ Annotation) {
		found = v
	})
	return found
}

func (b Basic) IsCaseSensitive() bool {
	return Q(ToNodes(b.Parameters)).IsCaseSensitive()
}

func (b Basic) Index() YesNoOnly {
	v := Q(ToNodes(b.Parameters)).yesNoOnlyValue(FieldIndex)
	if v == nil {
		return Yes
	}
	return *v
}

// A query is a tree of Nodes. We choose the type name Q so that external uses like query.Q do not stutter.
type Q []Node

func (q Q) String() string {
	return toString(q)
}

func (q Q) RegexpPatterns(field string) (values, negatedValues []string) {
	VisitField(q, field, func(visitedValue string, negated bool, _ Annotation) {
		if negated {
			negatedValues = append(negatedValues, visitedValue)
		} else {
			values = append(values, visitedValue)
		}
	})
	return values, negatedValues
}

func (q Q) StringValues(field string) (values, negatedValues []string) {
	VisitField(q, field, func(visitedValue string, negated bool, _ Annotation) {
		if negated {
			negatedValues = append(negatedValues, visitedValue)
		} else {
			values = append(values, visitedValue)
		}
	})
	return values, negatedValues
}

func (q Q) StringValue(field string) (value, negatedValue string) {
	VisitField(q, field, func(visitedValue string, negated bool, _ Annotation) {
		if negated {
			negatedValue = visitedValue
		} else {
			value = visitedValue
		}
	})
	return value, negatedValue
}

func (q Q) Values(field string) []*Value {
	var values []*Value
	if field == "" {
		VisitPattern(q, func(value string, _ bool, annotation Annotation) {
			values = append(values, q.valueToTypedValue(field, value, annotation.Labels)...)
		})
	} else {
		VisitField(q, field, func(value string, _ bool, _ Annotation) {
			values = append(values, q.valueToTypedValue(field, value, None)...)
		})
	}
	return values
}

func (q Q) Fields() map[string][]*Value {
	fields := make(map[string][]*Value)
	VisitPattern(q, func(value string, _ bool, _ Annotation) {
		fields[""] = q.Values("")
	})
	VisitParameter(q, func(field, _ string, _ bool, _ Annotation) {
		fields[field] = q.Values(field)
	})
	return fields
}

func (q Q) BoolValue(field string) bool {
	result := false
	VisitField(q, field, func(value string, _ bool, _ Annotation) {
		result, _ = parseBool(value) // err was checked during parsing and validation.
	})
	return result
}

func (q Q) Count() *int {
	var count *int
	VisitField(q, FieldCount, func(value string, _ bool, _ Annotation) {
		c, err := strconv.Atoi(value)
		if err != nil {
			panic(fmt.Sprintf("Value %q for count cannot be parsed as an int: %s", value, err))
		}
		count = &c
	})
	return count
}

func (q Q) Archived() *YesNoOnly {
	return q.yesNoOnlyValue(FieldArchived)
}

func (q Q) Fork() *YesNoOnly {
	return q.yesNoOnlyValue(FieldFork)
}

func (q Q) yesNoOnlyValue(field string) *YesNoOnly {
	var res *YesNoOnly
	VisitField(q, field, func(value string, _ bool, _ Annotation) {
		yno := ParseYesNoOnly(value)
		if yno == Invalid {
			panic(fmt.Sprintf("Invalid value %q for field %q", value, field))
		}
		res = &yno
	})
	return res
}

func (q Q) Timeout() *time.Duration {
	var timeout *time.Duration
	VisitField(q, FieldTimeout, func(value string, _ bool, _ Annotation) {
		t, err := time.ParseDuration(value)
		if err != nil {
			panic(fmt.Sprintf("Value %q for timeout cannot be parsed as an duration: %s", value, err))
		}
		timeout = &t
	})
	return timeout
}

func (q Q) IsCaseSensitive() bool {
	return q.BoolValue("case")
}

func (q Q) Repositories() (repos []string, negatedRepos []string) {
	VisitField(q, FieldRepo, func(value string, negated bool, _ Annotation) {
		if negated {
			negatedRepos = append(negatedRepos, value)
			return
		}
		repos = append(repos, value)
	})
	return repos, negatedRepos
}

func parseRegexpOrPanic(field, value string) *regexp.Regexp {
	r, err := regexp.Compile(value)
	if err != nil {
		panic(fmt.Sprintf("Value %s for field %s invalid regex: %s", field, value, err.Error()))
	}
	return r
}

// valueToTypedValue approximately preserves the field validation of our
// previous query processing. It does not check the validity of field negation
// or if the same field is specified more than once. This role is now performed
// by validate.go.
func (q Q) valueToTypedValue(field, value string, label labels) []*Value {
	switch field {
	case
		FieldDefault:
		if label.IsSet(Literal) {
			return []*Value{{String: &value}}
		}
		if label.IsSet(Regexp) {
			regexp, err := regexp.Compile(value)
			if err != nil {
				panic(fmt.Sprintf("Invariant broken: value must have been checked to be valid regexp. Error: %s", err))
			}
			return []*Value{{Regexp: regexp}}
		}
		// All patterns should have a label after parsing, but if not, treat the pattern as a string literal.
		return []*Value{{String: &value}}

	case
		FieldCase:
		b, _ := parseBool(value)
		return []*Value{{Bool: &b}}

	case
		FieldRepo, "r":
		return []*Value{{Regexp: parseRegexpOrPanic(field, value)}}

	case
		FieldRepoGroup, "g",
		FieldContext:
		return []*Value{{String: &value}}

	case
		FieldFile, "f":
		return []*Value{{Regexp: parseRegexpOrPanic(field, value)}}

	case
		FieldFork,
		FieldArchived,
		FieldLang, "l", "language",
		FieldType,
		FieldPatternType,
		FieldContent:
		return []*Value{{String: &value}}

	case FieldRepoHasFile:
		return []*Value{{Regexp: parseRegexpOrPanic(field, value)}}

	case
		FieldRepoHasCommitAfter,
		FieldBefore, "until",
		FieldAfter, "since":
		return []*Value{{String: &value}}

	case
		FieldAuthor,
		FieldCommitter,
		FieldMessage, "m", "msg":
		return []*Value{{Regexp: parseRegexpOrPanic(field, value)}}

	case
		FieldIndex,
		FieldCount,
		FieldTimeout,
		FieldCombyRule:
		return []*Value{{String: &value}}
	}
	return []*Value{{String: &value}}
}
