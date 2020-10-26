// Package query provides facilities for parsing and extracting
// information from search queries.
package query

import (
	"errors"
	"strings"

	"github.com/sourcegraph/sourcegraph/internal/search/query/syntax"
	"github.com/sourcegraph/sourcegraph/internal/search/query/types"
)

// All field names.
const (
	FieldDefault            = ""
	FieldCase               = "case"
	FieldRepo               = "repo"
	FieldRepoGroup          = "repogroup"
	FieldFile               = "file"
	FieldFork               = "fork"
	FieldArchived           = "archived"
	FieldLang               = "lang"
	FieldType               = "type"
	FieldRepoHasFile        = "repohasfile"
	FieldRepoHasCommitAfter = "repohascommitafter"
	FieldPatternType        = "patterntype"
	FieldContent            = "content"
	FieldVisibility         = "visibility"
	FieldRev                = "rev"

	// For diff and commit search only:
	FieldBefore    = "before"
	FieldAfter     = "after"
	FieldAuthor    = "author"
	FieldCommitter = "committer"
	FieldMessage   = "message"

	// Temporary experimental fields:
	FieldIndex     = "index"
	FieldCount     = "count"  // Searches that specify `count:` will fetch at least that number of results, or the full result set
	FieldStable    = "stable" // Forces search to return a stable result ordering (currently limited to file content matches).
	FieldMax       = "max"    // Deprecated alias for count
	FieldTimeout   = "timeout"
	FieldCombyRule = "rule"
)

var (
	regexpNegatableFieldType = types.FieldType{Literal: types.RegexpType, Quoted: types.RegexpType, Negatable: true}
	stringFieldType          = types.FieldType{Literal: types.StringType, Quoted: types.StringType}

	conf = types.Config{
		FieldTypes: map[string]types.FieldType{
			FieldDefault:     {Literal: types.RegexpType, Quoted: types.StringType},
			FieldCase:        {Literal: types.BoolType, Quoted: types.BoolType, Singular: true},
			FieldRepo:        regexpNegatableFieldType,
			FieldRepoGroup:   {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldFile:        regexpNegatableFieldType,
			FieldFork:        {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldArchived:    {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldLang:        {Literal: types.StringType, Quoted: types.StringType, Negatable: true},
			FieldType:        stringFieldType,
			FieldPatternType: {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldContent:     {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldVisibility:  {Literal: types.StringType, Quoted: types.StringType, Singular: true},

			FieldRepoHasFile:        regexpNegatableFieldType,
			FieldRepoHasCommitAfter: {Literal: types.StringType, Quoted: types.StringType, Singular: true},

			FieldBefore:    stringFieldType,
			FieldAfter:     stringFieldType,
			FieldAuthor:    regexpNegatableFieldType,
			FieldCommitter: regexpNegatableFieldType,
			FieldMessage:   regexpNegatableFieldType,

			// Experimental fields:
			FieldIndex:     {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldCount:     {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldStable:    {Literal: types.BoolType, Quoted: types.BoolType, Singular: true},
			FieldMax:       {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldTimeout:   {Literal: types.StringType, Quoted: types.StringType, Singular: true},
			FieldCombyRule: {Literal: types.StringType, Quoted: types.StringType, Singular: true},
		},
		FieldAliases: map[string]string{
			"r":        FieldRepo,
			"g":        FieldRepoGroup,
			"f":        FieldFile,
			"l":        FieldLang,
			"language": FieldLang,
			"since":    FieldAfter,
			"until":    FieldBefore,
			"m":        FieldMessage,
			"msg":      FieldMessage,
		},
	}
)

// A Query is the typechecked representation of a search query.
type Query struct {
	conf *types.Config // the typechecker config used to produce this query

	types.Fields // the query fields
}

func Parse(input string) (syntax.ParseTree, error) {
	parseTree, err := syntax.Parse(input)
	if err != nil {
		return nil, err
	}

	// We want to make query fields case insensitive
	for _, expr := range parseTree {
		expr.Field = strings.ToLower(expr.Field)
	}
	return parseTree, nil
}

func Check(parseTree syntax.ParseTree) (QueryInfo, error) {
	checkedFields, err := conf.Check(parseTree)
	if err != nil {
		return nil, err
	}
	query := &Query{conf: &conf, Fields: *checkedFields}
	return &OrdinaryQuery{Query: query}, nil
}

// ParseAndCheck parses and typechecks a search query using the default
// query type configuration.
func ParseAndCheck(input string) (QueryInfo, error) {
	parseTree, err := Parse(input)
	if err != nil {
		return nil, err
	}

	checkedQuery, err := Check(parseTree)
	if err != nil {
		return nil, err
	}

	ordinaryQuery, ok := checkedQuery.(*OrdinaryQuery)
	if !ok {
		return nil, errors.New("Check failed: Expected OrdinaryQuery")
	}
	ordinaryQuery.parseTree = parseTree
	return ordinaryQuery, err
}

func processSearchPattern(q QueryInfo) string {
	var pieces []string
	for _, v := range q.Values(FieldDefault) {
		if piece := v.ToString(); piece != "" {
			pieces = append(pieces, piece)
		}
	}
	return strings.Join(pieces, " ")
}

type ValidationError struct {
	Msg string
}

func (e *ValidationError) Error() string {
	return e.Msg
}

// Validate validates legal combinations of fields and search patterns of a
// successfully parsed query.
func Validate(q QueryInfo, searchType SearchType) error {
	if searchType == SearchTypeStructural {
		if q.Fields()[FieldCase] != nil {
			return errors.New(`the parameter "case:" is not valid for structural search, matching is always case-sensitive`)
		}
		if q.Fields()[FieldType] != nil && processSearchPattern(q) != "" {
			return errors.New(`the parameter "type:" is not valid for structural search, search is always performed on file content`)
		}
	}
	return nil
}

// Process is a top level convenience function for processing a raw string into
// a validated and type checked query, and the parse tree of the raw string.
func Process(queryString string, searchType SearchType) (QueryInfo, error) {
	parseTree, err := Parse(queryString)
	if err != nil {
		return nil, err
	}

	query, err := Check(parseTree)
	if err != nil {
		return nil, err
	}

	err = Validate(query, searchType)
	if err != nil {
		return nil, err
	}

	ordinaryQuery, ok := query.(*OrdinaryQuery)
	if !ok {
		return nil, errors.New("Check failed: Expected OrdinaryQuery")
	}
	ordinaryQuery.parseTree = parseTree

	return ordinaryQuery, nil
}

// parseAndCheck is preserved for testing custom Configs only.
func parseAndCheck(conf *types.Config, input string) (*Query, error) {
	parseTree, err := syntax.Parse(input)
	if err != nil {
		return nil, err
	}

	// We want to make query fields case insensitive
	for _, expr := range parseTree {
		expr.Field = strings.ToLower(expr.Field)
	}

	checkedQuery, err := conf.Check(parseTree)
	if err != nil {
		return nil, err
	}
	return &Query{conf: conf, Fields: *checkedQuery}, nil
}

// BoolValue returns the last boolean value (yes/no) for the field. For example, if the query is
// "foo:yes foo:no foo:yes", then the last boolean value for the "foo" field is true ("yes"). The
// default boolean value is false.
func (q *Query) BoolValue(field string) bool {
	for _, v := range q.Fields[field] {
		if v.Bool != nil {
			return *v.Bool
		}
	}
	return false // default
}

// IsCaseSensitive reports whether the query's expressions are matched
// case sensitively.
func (q *Query) IsCaseSensitive() bool {
	return q.BoolValue(FieldCase)
}

// Values returns the values for the given field.
func (q *Query) Values(field string) []*types.Value {
	if _, ok := q.conf.FieldTypes[field]; !ok {
		panic("no such field: " + field)
	}
	return q.Fields[field]
}

// RegexpPatterns returns the regexp pattern source strings for the given field.
// If the field is not recognized or it is not always regexp-typed, it panics.
func (q *Query) RegexpPatterns(field string) (values, negatedValues []string) {
	fieldType, ok := q.conf.FieldTypes[field]
	if !ok {
		panic("no such field: " + field)
	}
	if fieldType.Literal != types.RegexpType || fieldType.Quoted != types.RegexpType {
		panic("field is not always regexp-typed: " + field)
	}

	for _, v := range q.Fields[field] {
		s := v.ToString()
		if v.Not() {
			negatedValues = append(negatedValues, s)
		} else {
			values = append(values, s)
		}
	}
	return
}

// StringValues returns the string values for the given field. If the field is
// not recognized or it is not always string-typed, it panics.
func (q *Query) StringValues(field string) (values, negatedValues []string) {
	fieldType, ok := q.conf.FieldTypes[field]
	if !ok {
		panic("no such field: " + field)
	}
	if fieldType.Literal != types.StringType || fieldType.Quoted != types.StringType {
		panic("field is not always string-typed: " + field)
	}

	for _, v := range q.Fields[field] {
		if v.Not() {
			negatedValues = append(negatedValues, *v.String)
		} else {
			values = append(values, *v.String)
		}
	}
	return
}

// StringValue returns the string value for the given field.
// It panics if the field is not recognized, it is not always string-typed, or it is not singular.
func (q *Query) StringValue(field string) (value, negatedValue string) {
	fieldType, ok := q.conf.FieldTypes[field]
	if !ok {
		panic("no such field: " + field)
	}
	if fieldType.Literal != types.StringType || fieldType.Quoted != types.StringType {
		panic("field is not always string-typed: " + field)
	}
	if !fieldType.Singular {
		panic("field is not singular: " + field)
	}
	if len(q.Fields[field]) == 0 {
		return "", ""
	}
	v := q.Fields[field][0]
	if v.Not() {
		return "", *v.String
	}
	return *v.String, ""
}
