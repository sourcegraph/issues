package query

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func collectLabels(nodes []Node) (result labels) {
	for _, node := range nodes {
		switch v := node.(type) {
		case Operator:
			result |= v.Annotation.Labels
			result |= collectLabels(v.Operands)
		case Pattern:
			result |= v.Annotation.Labels
		}
	}
	return result
}

func heuristicLabels(nodes []Node) string {
	labels := collectLabels(nodes)
	return strings.Join(labels.String(), ",")
}

func TestParseParameterList(t *testing.T) {
	cases := []struct {
		Name       string
		Input      string
		Want       string
		WantLabels labels
		WantRange  string
	}{
		{
			Name:       "Normal field:value",
			Input:      `file:README.md`,
			Want:       `{"field":"file","value":"README.md","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":14}}`,
			WantLabels: None,
		},
		{
			Name:       "Normal field:value with trailing space",
			Input:      `file:README.md    `,
			Want:       `{"field":"file","value":"README.md","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":14}}`,
			WantLabels: None,
		},
		{
			Name:       "First char is colon",
			Input:      `:foo`,
			Want:       `{"value":":foo","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":4}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Last char is colon",
			Input:      `foo:`,
			Want:       `{"value":"foo:","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":4}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Match first colon",
			Input:      `file:bar:baz`,
			Want:       `{"field":"file","value":"bar:baz","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":12}}`,
			WantLabels: None,
		},
		{
			Name:       "No field, start with minus",
			Input:      `-:foo`,
			Want:       `{"value":"-:foo","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":5}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Minus prefix on field",
			Input:      `-file:README.md`,
			Want:       `{"field":"file","value":"README.md","negated":true}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":15}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "NOT prefix on file",
			Input:      `NOT file:README.md`,
			Want:       `{"field":"file","value":"README.md","negated":true}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":18}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "NOT prefix on unsupported key-value pair",
			Input:      `NOT foo:bar`,
			Want:       `{"value":"foo:bar","negated":true}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":11}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "NOT prefix on content",
			Input:      `NOT content:bar`,
			Want:       `{"field":"content","value":"bar","negated":true}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":15}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Double NOT",
			Input:      `NOT NOT`,
			Want:       `{"value":"NOT","negated":true}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":7}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Double minus prefix on field",
			Input:      `--foo:bar`,
			Want:       `{"value":"--foo:bar","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":9}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Minus in the middle is not a valid field",
			Input:      `fie-ld:bar`,
			Want:       `{"value":"fie-ld:bar","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":10}}`,
			WantLabels: Regexp,
		},
		{
			Name:       "Preserve escaped whitespace",
			Input:      `a\ pattern`,
			Want:       `{"value":"a\\ pattern","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":10}}`,
			WantLabels: Regexp,
		},
		{
			Input:      `"quoted"`,
			Want:       `{"value":"quoted","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":8}}`,
			WantLabels: Literal | Quoted,
		},
		{
			Input:      `'\''`,
			Want:       `{"value":"'","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":4}}`,
			WantLabels: Literal | Quoted,
		},
		{
			Input:      `foo.*bar(`,
			Want:       `{"value":"foo.*bar(","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":9}}`,
			WantLabels: Regexp | HeuristicDanglingParens,
		},
		{
			Input:      `/a regex pattern/`,
			Want:       `{"value":"a regex pattern","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":17}}`,
			WantLabels: Regexp,
		},
		{
			Input:      `Search()\(`,
			Want:       `{"value":"Search()\\(","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":10}}`,
			WantLabels: Regexp,
		},
		{
			Input:      `Search(xxx)\(`,
			Want:       `{"value":"Search(xxx)\\(","negated":false}`,
			WantRange:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":13}}`,
			WantLabels: Regexp,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			parser := &parser{buf: []byte(tt.Input), heuristics: parensAsPatterns | allowDanglingParens}
			result, err := parser.parseLeavesRegexp()
			if err != nil {
				t.Fatal(fmt.Sprintf("Unexpected error: %s", err))
			}
			resultNode := result[0]
			got, _ := json.Marshal(resultNode)
			// Check parsed values.
			if diff := cmp.Diff(tt.Want, string(got)); diff != "" {
				t.Error(diff)
			}
			// Check ranges.
			switch n := resultNode.(type) {
			case Pattern:
				rangeStr := n.Annotation.Range.String()
				if diff := cmp.Diff(tt.WantRange, rangeStr); diff != "" {
					t.Error(diff)
				}
			case Parameter:
				rangeStr := n.Annotation.Range.String()
				if diff := cmp.Diff(tt.WantRange, rangeStr); diff != "" {
					t.Error(diff)
				}
			}
			// Check labels.
			if patternNode, ok := resultNode.(Pattern); ok {
				if diff := cmp.Diff(tt.WantLabels, patternNode.Annotation.Labels); diff != "" {
					t.Error(diff)
				}
			}
		})
	}
}

func TestScanField(t *testing.T) {
	type value struct {
		Field   string
		Negated bool
		Advance int
	}
	cases := []struct {
		Input   string
		Negated bool
		Want    value
	}{
		// Valid field.
		{
			Input: "repo:foo",
			Want: value{
				Field:   "repo",
				Advance: 5,
			},
		},
		{
			Input: "RepO:foo",
			Want: value{
				Field:   "RepO",
				Advance: 5,
			},
		},
		{
			Input: "after:",
			Want: value{
				Field:   "after",
				Advance: 6,
			},
		},
		{
			Input: "-repo:",
			Want: value{
				Field:   "repo",
				Negated: true,
				Advance: 6,
			},
		},
		// Invalid field.
		{
			Input: "",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: "-",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: "-:",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: ":",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: "??:foo",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: "repo",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: "-repo",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: "--repo:",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
		{
			Input: ":foo",
			Want: value{
				Field:   "",
				Advance: 0,
			},
		},
	}
	for _, c := range cases {
		t.Run("scan field", func(t *testing.T) {
			gotField, gotNegated, gotAdvance := ScanField([]byte(c.Input))
			if diff := cmp.Diff(c.Want, value{gotField, gotNegated, gotAdvance}); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func parseAndOrGrammar(in string) ([]Node, error) {
	if strings.TrimSpace(in) == "" {
		return nil, nil
	}
	parser := &parser{
		buf:        []byte(in),
		leafParser: SearchTypeRegex,
	}
	nodes, err := parser.parseOr()
	if err != nil {
		return nil, err
	}
	if parser.balanced != 0 {
		return nil, errors.New("unbalanced expression")
	}
	return newOperator(nodes, And), nil
}

func TestParse(t *testing.T) {
	type relation string         // a relation for comparing test outputs of queries parsed according to grammar and heuristics.
	const Same relation = "Same" // a constant that says heuristic output is interpreted the same as the grammar spec.
	type Spec = relation         // constructor for expected output of the grammar spec without heuristics.
	type Diff = relation         // constructor for expected heuristic output when different to the grammar spec.

	cases := []struct {
		Name          string
		Input         string
		WantGrammar   relation
		WantHeuristic relation
	}{
		{
			Name:          "Empty string",
			Input:         "",
			WantGrammar:   "",
			WantHeuristic: Same,
		},
		{
			Name:          "Whitespace",
			Input:         "             ",
			WantGrammar:   "",
			WantHeuristic: Same,
		},
		{
			Name:          "Single",
			Input:         "a",
			WantGrammar:   `"a"`,
			WantHeuristic: Same,
		},
		{
			Name:          "Whitespace basic",
			Input:         "a b",
			WantGrammar:   `(concat "a" "b")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Basic",
			Input:         "a and b and c",
			WantGrammar:   `(and "a" "b" "c")`,
			WantHeuristic: Same,
		},
		{
			Input:         "(f(x)oo((a|b))bar)",
			WantGrammar:   Spec(`(concat "f" "x" "oo" "a|b" "bar")`),
			WantHeuristic: Diff(`"(f(x)oo((a|b))bar)"`),
		},
		{
			Input:         "aorb",
			WantGrammar:   `"aorb"`,
			WantHeuristic: Same,
		},
		{
			Input:         "aANDb",
			WantGrammar:   `"aANDb"`,
			WantHeuristic: Same,
		},
		{
			Input:         "a oror b",
			WantGrammar:   `(concat "a" "oror" "b")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Reduced complex query mixed caps",
			Input:         "a and b AND c or d and (e OR f) g h i or j",
			WantGrammar:   `(or (and "a" "b" "c") (and "d" (concat (or "e" "f") "g" "h" "i")) "j")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Basic reduced complex query",
			Input:         "a and b or c and d or e",
			WantGrammar:   `(or (and "a" "b") (and "c" "d") "e")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Reduced complex query, reduction over parens",
			Input:         "(a and b or c and d) or e",
			WantGrammar:   `(or (and "a" "b") (and "c" "d") "e")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Reduced complex query, nested 'or' trickles up",
			Input:         "(a and b or c) or d",
			WantGrammar:   `(or (and "a" "b") "c" "d")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Reduced complex query, nested nested 'or' trickles up",
			Input:         "(a and b or (c and d or f)) or e",
			WantGrammar:   `(or (and "a" "b") (and "c" "d") "f" "e")`,
			WantHeuristic: Same,
		},
		{
			Name:          "No reduction on precedence defined by parens",
			Input:         "(a and (b or c) and d) or e",
			WantGrammar:   `(or (and "a" (or "b" "c") "d") "e")`,
			WantHeuristic: Same,
		},
		{
			Name:          "Paren reduction over operators",
			Input:         "(((a b c))) and d",
			WantGrammar:   Spec(`(and (concat "a" "b" "c") "d")`),
			WantHeuristic: Diff(`(and "(((a b c)))" "d")`),
		},
		// Partition parameters and concatenated patterns.
		{
			Input:         "a (b and c) d",
			WantGrammar:   `(concat "a" (and "b" "c") "d")`,
			WantHeuristic: Same,
		},
		{
			Input:         "(a b c) and (d e f) and (g h i)",
			WantGrammar:   Spec(`(and (concat "a" "b" "c") (concat "d" "e" "f") (concat "g" "h" "i"))`),
			WantHeuristic: `(and "(a b c)" "(d e f)" "(g h i)")`,
		},
		{
			Input:         "(a) repo:foo (b)",
			WantGrammar:   Spec(`(and "repo:foo" (concat "a" "b"))`),
			WantHeuristic: Diff(`(and "repo:foo" (concat "(a)" "(b)"))`),
		},
		{
			Input:         "repo:foo func( or func(.*)",
			WantGrammar:   Spec(`expected operand at 15`),
			WantHeuristic: Diff(`(and "repo:foo" (or "func(" "func(.*)"))`),
		},
		{
			Input:         "repo:foo main { and bar {",
			WantGrammar:   Spec(`(and (and "repo:foo" (concat "main" "{")) (concat "bar" "{"))`),
			WantHeuristic: Diff(`(and "repo:foo" (concat "main" "{") (concat "bar" "{"))`),
		},
		{
			Input:         "a b (repo:foo c d)",
			WantGrammar:   `(concat "a" "b" (and "repo:foo" (concat "c" "d")))`,
			WantHeuristic: Same,
		},
		{
			Input:         "a b (c d repo:foo)",
			WantGrammar:   `(concat "a" "b" (and "repo:foo" (concat "c" "d")))`,
			WantHeuristic: Same,
		},
		{
			Input:         "a repo:b repo:c (d repo:e repo:f)",
			WantGrammar:   `(and "repo:b" "repo:c" (concat "a" (and "repo:e" "repo:f" "d")))`,
			WantHeuristic: Same,
		},
		{
			Input:         "a repo:b repo:c (repo:e repo:f (repo:g repo:h))",
			WantGrammar:   `(and "repo:b" "repo:c" "repo:e" "repo:f" "repo:g" "repo:h" "a")`,
			WantHeuristic: Same,
		},
		{
			Input:         "a repo:b repo:c (repo:e repo:f (repo:g repo:h)) b",
			WantGrammar:   `(and "repo:b" "repo:c" "repo:e" "repo:f" "repo:g" "repo:h" (concat "a" "b"))`,
			WantHeuristic: Same,
		},
		{
			Input:         "a repo:b repo:c (repo:e repo:f (repo:g repo:h b)) ",
			WantGrammar:   `(and "repo:b" "repo:c" (concat "a" (and "repo:e" "repo:f" "repo:g" "repo:h" "b")))`,
			WantHeuristic: Same,
		},
		{
			Input:         "(repo:foo a (repo:bar b (repo:qux c)))",
			WantGrammar:   `(and "repo:foo" (concat "a" (and "repo:bar" (concat "b" (and "repo:qux" "c")))))`,
			WantHeuristic: Same,
		},
		{
			Input:         "a repo:b repo:c (d repo:e repo:f e)",
			WantGrammar:   `(and "repo:b" "repo:c" (concat "a" (and "repo:e" "repo:f" (concat "d" "e"))))`,
			WantHeuristic: Same,
		},
		// Keywords as patterns.
		{
			Input:         "a or",
			WantGrammar:   `(concat "a" "or")`,
			WantHeuristic: Same,
		},
		{
			Input:         "or",
			WantGrammar:   `"or"`,
			WantHeuristic: Same,
		},
		{
			Input:         "or or or",
			WantGrammar:   `(or "or" "or")`,
			WantHeuristic: Same,
		},
		{
			Input:         "and and andand or oror",
			WantGrammar:   `(or (and "and" "andand") "oror")`,
			WantHeuristic: Same,
		},
		// Errors.
		{
			Name:          "Unbalanced",
			Input:         "(foo) (bar",
			WantGrammar:   Spec("unbalanced expression"),
			WantHeuristic: Diff(`(concat "(foo)" "(bar")`),
		},
		{
			Name:          "Illegal expression on the right",
			Input:         "a or or b",
			WantGrammar:   "expected operand at 5",
			WantHeuristic: Same,
		},
		{
			Name:          "Illegal expression on the right, mixed operators",
			Input:         "a and OR",
			WantGrammar:   `(and "a" "OR")`,
			WantHeuristic: Same,
		},
		{
			Input:         "repo:foo or or or",
			WantGrammar:   "expected operand at 12",
			WantHeuristic: Same,
		},
		// Reduction.
		{
			Name:          "paren reduction with ands",
			Input:         "(a and b) and (c and d)",
			WantGrammar:   `(and "a" "b" "c" "d")`,
			WantHeuristic: Same,
		},
		{
			Name:          "paren reduction with ors",
			Input:         "(a or b) or (c or d)",
			WantGrammar:   `(or "a" "b" "c" "d")`,
			WantHeuristic: Same,
		},
		{
			Name:          "nested paren reduction with whitespace",
			Input:         "(((a b c))) d",
			WantGrammar:   Spec(`(concat "a" "b" "c" "d")`),
			WantHeuristic: Diff(`(concat "(((a b c)))" "d")`),
		},
		{
			Name:          "left paren reduction with whitespace",
			Input:         "(a b) c d",
			WantGrammar:   Spec(`(concat "a" "b" "c" "d")`),
			WantHeuristic: Diff(`(concat "(a b)" "c" "d")`),
		},
		{
			Name:          "right paren reduction with whitespace",
			Input:         "a b (c d)",
			WantGrammar:   Spec(`(concat "a" "b" "c" "d")`),
			WantHeuristic: Diff(`(concat "a" "b" "(c d)")`),
		},
		{
			Name:          "grouped paren reduction with whitespace",
			Input:         "(a b) (c d)",
			WantGrammar:   Spec(`(concat "a" "b" "c" "d")`),
			WantHeuristic: Diff(`(concat "(a b)" "(c d)")`),
		},
		{
			Name:          "multiple grouped paren reduction with whitespace",
			Input:         "(a b) (c d) (e f)",
			WantGrammar:   Spec(`(concat "a" "b" "c" "d" "e" "f")`),
			WantHeuristic: Diff(`(concat "(a b)" "(c d)" "(e f)")`),
		},
		{
			Name:          "interpolated grouped paren reduction",
			Input:         "(a b) c d (e f)",
			WantGrammar:   Spec(`(concat "a" "b" "c" "d" "e" "f")`),
			WantHeuristic: Diff(`(concat "(a b)" "c" "d" "(e f)")`),
		},
		{
			Name:          "mixed interpolated grouped paren reduction",
			Input:         "(a and b and (z or q)) and (c and d) and (e and f)",
			WantGrammar:   `(and "a" "b" (or "z" "q") "c" "d" "e" "f")`,
			WantHeuristic: Same,
		},
		// Parentheses.
		{
			Name:          "empty paren",
			Input:         "()",
			WantGrammar:   Spec(`""`),
			WantHeuristic: Diff(`"()"`),
		},
		{
			Name:          "paren inside contiguous string",
			Input:         "foo()bar",
			WantGrammar:   Spec(`(concat "foo" "bar")`),
			WantHeuristic: Diff(`"foo()bar"`),
		},
		{
			Name:          "paren inside contiguous string",
			Input:         "(x and regex(s)?)",
			WantGrammar:   Spec(`(and "x" (concat "regex" "s" "?"))`),
			WantHeuristic: Diff(`(and "x" "regex(s)?")`),
		},
		{
			Name:          "paren containing whitespace inside contiguous string",
			Input:         "foo(   )bar",
			WantGrammar:   Spec(`(concat "foo" "bar")`),
			WantHeuristic: Diff(`"foo(   )bar"`),
		},
		{
			Name:          "nested empty paren",
			Input:         "(x())",
			WantGrammar:   Spec(`"x"`),
			WantHeuristic: Diff(`"(x())"`),
		},
		{
			Name:          "interpolated nested empty paren",
			Input:         "(()x(  )(())())",
			WantGrammar:   Spec(`"x"`),
			WantHeuristic: Diff(`"(()x(  )(())())"`),
		},
		{
			Name:          "empty paren on or",
			Input:         "() or ()",
			WantGrammar:   Spec(`""`),
			WantHeuristic: Diff(`(or "()" "()")`),
		},
		{
			Name:          "empty left paren on or",
			Input:         "() or (x)",
			WantGrammar:   Spec(`"x"`),
			WantHeuristic: Diff(`(or "()" "(x)")`),
		},
		{
			Name:          "complex interpolated nested empty paren",
			Input:         "(()x(  )(y or () or (f))())",
			WantGrammar:   Spec(`(concat "x" (or "y" "f"))`),
			WantHeuristic: Diff(`(concat "()" "x" "()" (or "y" "()" "(f)") "()")`),
		},
		{
			Name:          "disable parens as patterns heuristic if containing recognized operator",
			Input:         "(() or ())",
			WantGrammar:   Spec(`""`),
			WantHeuristic: Diff(`(or "()" "()")`),
		},
		// Escaping.
		{
			Input:         `\(\)`,
			WantGrammar:   `"\\(\\)"`,
			WantHeuristic: Same,
		},
		{
			Input:         `\( \) ()`,
			WantGrammar:   Spec(`(concat "\\(" "\\)")`),
			WantHeuristic: Diff(`(concat "\\(" "\\)" "()")`),
		},
		{
			Input:         `\ `,
			WantGrammar:   `"\\ "`,
			WantHeuristic: Same,
		},
		{
			Input:         `\  \ `,
			WantGrammar:   Spec(`(concat "\\ " "\\ ")`),
			WantHeuristic: Diff(`(concat "\\ " "\\ ")`),
		},
		// Dangling parentheses heuristic.
		{
			Input:         `(`,
			WantGrammar:   Spec(`expected operand at 1`),
			WantHeuristic: Diff(`"("`),
		},
		{
			Input:         `)(())(`,
			WantGrammar:   Spec(`unbalanced expression`),
			WantHeuristic: Diff(`"(())("`),
		},
		{
			Input:         `foo( and bar(`,
			WantGrammar:   Spec(`expected operand at 5`),
			WantHeuristic: Diff(`(and "foo(" "bar(")`),
		},
		{
			Input:         `repo:foo foo( or bar(`,
			WantGrammar:   Spec(`expected operand at 14`),
			WantHeuristic: Diff(`(and "repo:foo" (or "foo(" "bar("))`),
		},
		{
			Input:         `(a or (b and )) or d)`,
			WantGrammar:   Spec(`unbalanced expression`),
			WantHeuristic: Diff(`(or "(a" (and "(b" ")") "d)")`),
		},
		// Quotes and escape sequences.
		{
			Input:         `"`,
			WantGrammar:   `"\""`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo' bar'`,
			WantGrammar:   `(and "repo:foo'" "bar'")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:'foo' 'bar'`,
			WantGrammar:   `(and "repo:foo" "bar")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:"foo" "bar"`,
			WantGrammar:   `(and "repo:foo" "bar")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:"foo bar" "foo bar"`,
			WantGrammar:   `(and "repo:foo bar" "foo bar")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:"fo\"o" "bar"`,
			WantGrammar:   Spec(`(and "repo:fo\"o" "bar")`),
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo /b\/ar/`,
			WantGrammar:   `(and "repo:foo" "b/ar")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo /a/file/path`,
			WantGrammar:   `(and "repo:foo" "/a/file/path")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo /a/file/path/`,
			WantGrammar:   `(and "repo:foo" "/a/file/path/")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo /a/ /another/path/`,
			WantGrammar:   `(and "repo:foo" (concat "a" "/another/path/"))`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo /\s+b\d+ar/ `,
			WantGrammar:   `(and "repo:foo" "\\s+b\\d+ar")`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo /bar/ `,
			WantGrammar:   `(and "repo:foo" "bar")`,
			WantHeuristic: Same,
		},
		{
			Input:         `\t\r\n`,
			WantGrammar:   `"\\t\\r\\n"`,
			WantHeuristic: Same,
		},
		{
			Input:         `repo:foo\ bar \:\\`,
			WantGrammar:   `(and "repo:foo\\ bar" "\\:\\\\")`,
			WantHeuristic: Same,
		},
		{
			Input:         `a file:\.(ts(?:(?:)|x)|js(?:(?:)|x))(?m:$)`,
			WantGrammar:   `(and "file:\\.(ts(?:(?:)|x)|js(?:(?:)|x))(?m:$)" "a")`,
			WantHeuristic: Same,
		},
		{
			Input:         `(file:(a) file:(b))`,
			WantGrammar:   `(and "file:(a)" "file:(b)")`,
			WantHeuristic: Same,
		},
		{
			Input:         `(repohascommitafter:"7 days")`,
			WantGrammar:   `"repohascommitafter:7 days"`,
			WantHeuristic: Same,
		},
		{
			Input:         `(foo repohascommitafter:"7 days")`,
			WantGrammar:   `(and "repohascommitafter:7 days" "foo")`,
			WantHeuristic: Same,
		},
		// Fringe tests cases at the boundary of heuristics and invalid syntax.
		{
			Input:         `(0(F)(:())(:())(<0)0()`,
			WantGrammar:   Spec(`unbalanced expression`),
			WantHeuristic: `"(0(F)(:())(:())(<0)0()"`,
		},
		// The space-looking character below is U+00A0.
		{
			Input:         `00 (000)`,
			WantGrammar:   `(concat "00" "000")`,
			WantHeuristic: `(concat "00" "(000)")`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			check := func(result []Node, err error, want string) {
				var resultStr []string
				if err != nil {
					if diff := cmp.Diff(want, err.Error()); diff != "" {
						t.Fatal(diff)
					}
					return
				}
				for _, node := range result {
					resultStr = append(resultStr, node.String())
				}
				got := strings.Join(resultStr, " ")
				if diff := cmp.Diff(want, got); diff != "" {
					t.Error(diff)
				}
			}
			var result []Node
			var err error
			result, err = parseAndOrGrammar(tt.Input) // Parse without heuristic.
			check(result, err, string(tt.WantGrammar))
			result, err = ParseAndOr(tt.Input, SearchTypeRegex)
			if tt.WantHeuristic == Same {
				check(result, err, string(tt.WantGrammar))
			} else {
				check(result, err, string(tt.WantHeuristic))
			}
		})
	}
}

func TestScanDelimited(t *testing.T) {
	type result struct {
		Value  string
		Count  int
		ErrMsg string
	}

	cases := []struct {
		name      string
		input     string
		delimiter rune
		want      result
	}{
		{
			input:     `""`,
			delimiter: '"',
			want:      result{Value: "", Count: 2, ErrMsg: ""},
		},
		{
			input:     `"a"`,
			delimiter: '"',
			want:      result{Value: `a`, Count: 3, ErrMsg: ""},
		},
		{
			input:     `"\""`,
			delimiter: '"',
			want:      result{Value: `"`, Count: 4, ErrMsg: ""},
		},
		{
			input:     `"\\""`,
			delimiter: '"',
			want:      result{Value: `\`, Count: 4, ErrMsg: ""},
		},
		{
			input:     `"\\\"`,
			delimiter: '"',
			want:      result{Value: "", Count: 5, ErrMsg: `unterminated literal: expected "`},
		},
		{
			input:     `"\\\""`,
			delimiter: '"',
			want:      result{Value: `\"`, Count: 6, ErrMsg: ""},
		},
		{
			input:     `"a`,
			delimiter: '"',
			want:      result{Value: "", Count: 2, ErrMsg: `unterminated literal: expected "`},
		},
		{
			input:     `"\?"`,
			delimiter: '"',
			want:      result{Value: "", Count: 3, ErrMsg: `unrecognized escape sequence`},
		},
		{
			name:      "panic",
			input:     `a"`,
			delimiter: '"',
			want:      result{},
		},
		{
			input:     `/\//`,
			delimiter: '/',
			want:      result{Value: "/", Count: 4, ErrMsg: ""},
		},
	}

	for _, tt := range cases {
		if tt.name == "panic" {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for ScanDelimited")
				}
			}()
			_, _, _ = ScanDelimited([]byte(tt.input), true, tt.delimiter)
		}

		t.Run(tt.name, func(t *testing.T) {
			value, count, err := ScanDelimited([]byte(tt.input), true, tt.delimiter)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			got := result{value, count, errMsg}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestMergePatterns(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{
			input: "foo()bar",
			want:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":8}}`,
		},
		{
			input: "()bar",
			want:  `{"start":{"line":0,"column":0},"end":{"line":0,"column":5}}`,
		},
	}

	for _, tt := range cases {
		t.Run("merge pattern", func(t *testing.T) {
			p := &parser{buf: []byte(tt.input), heuristics: parensAsPatterns}
			nodes, err := p.parseLeavesRegexp()
			got := nodes[0].(Pattern).Annotation.Range.String()
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestMatchUnaryKeyword(t *testing.T) {
	tests := []struct {
		in   string
		pos  int
		want bool
	}{
		{
			in:   "NOT bar",
			pos:  0,
			want: true,
		},
		{
			in:   "foo NOT bar",
			pos:  4,
			want: true,
		},
		{
			in:   "foo NOT",
			pos:  4,
			want: false,
		},
		{
			in:   "fooNOT bar",
			pos:  3,
			want: false,
		},
		{
			in:   "NOTbar",
			pos:  0,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			p := &parser{buf: []byte(tt.in), pos: tt.pos}
			if got := p.matchUnaryKeyword("NOT"); got != tt.want {
				t.Errorf("matchUnaryKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAndOrLiteral(t *testing.T) {
	cases := []struct {
		Input      string
		Want       string
		WantLabels string
		WantError  string
	}{
		{
			Input:      "()",
			Want:       `"()"`,
			WantLabels: "HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      `"`,
			Want:       `"\""`,
			WantLabels: "Literal",
		},
		{
			Input:      `""`,
			Want:       `"\"\""`,
			WantLabels: "Literal",
		},
		{
			Input:      "(",
			Want:       `"("`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      "repo:foo foo( or bar(",
			Want:       `(and "repo:foo" (or "foo(" "bar("))`,
			WantLabels: "HeuristicHoisted,Literal",
		},
		{
			Input:      "x or",
			Want:       `(concat "x" "or")`,
			WantLabels: "Literal",
		},
		{
			Input:      "repo:foo (x",
			Want:       `(and "repo:foo" "(x")`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      "(x or bar() )",
			Want:       `(or "x" "bar()")`,
			WantLabels: "Literal",
		},
		{
			Input:      "(x",
			Want:       `"(x"`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      "x or (x",
			Want:       `(or "x" "(x")`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,Literal",
		},
		{
			Input:      "(y or (z",
			Want:       `(or "(y" "(z")`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,Literal",
		},
		{
			Input:      "repo:foo (lisp)",
			Want:       `(and "repo:foo" "(lisp)")`,
			WantLabels: "HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      "repo:foo (lisp lisp())",
			Want:       `(and "repo:foo" "(lisp lisp())")`,
			WantLabels: "HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      "repo:foo (lisp or lisp)",
			Want:       `(and "repo:foo" (or "lisp" "lisp"))`,
			WantLabels: "Literal",
		},
		{
			Input:      "repo:foo (lisp or lisp())",
			Want:       `(and "repo:foo" (or "lisp" "lisp()"))`,
			WantLabels: "Literal",
		},
		{
			Input:      "repo:foo (lisp or lisp())",
			Want:       `(and "repo:foo" (or "lisp" "lisp()"))`,
			WantLabels: "Literal",
		},
		{
			Input:      "repo:foo (lisp or lisp()",
			Want:       `(and "repo:foo" (or "(lisp" "lisp()"))`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,Literal",
		},
		{
			Input:      "(y or bar())",
			Want:       `(or "y" "bar()")`,
			WantLabels: "Literal",
		},
		{
			Input:      "((x or bar(",
			Want:       `(or "((x" "bar(")`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,Literal",
		},
		{
			Input:      "",
			Want:       "",
			WantLabels: "None",
		},
		{
			Input:      " ",
			Want:       "",
			WantLabels: "None",
		},
		{
			Input:      "  ",
			Want:       "",
			WantLabels: "None",
		},
		{
			Input:      "a",
			Want:       `"a"`,
			WantLabels: "Literal",
		},
		{
			Input:      " a",
			Want:       `"a"`,
			WantLabels: "Literal",
		},
		{
			Input:      `a `,
			Want:       `"a"`,
			WantLabels: "Literal",
		},
		{
			Input:      ` a b`,
			Want:       `(concat "a" "b")`,
			WantLabels: "Literal",
		},
		{
			Input:      `a  b`,
			Want:       `(concat "a" "b")`,
			WantLabels: "Literal",
		},
		{
			Input:      `:`,
			Want:       `":"`,
			WantLabels: "Literal",
		},
		{
			Input:      `:=`,
			Want:       `":="`,
			WantLabels: "Literal",
		},
		{
			Input:      `:= range`,
			Want:       `(concat ":=" "range")`,
			WantLabels: "Literal",
		},
		{
			Input:      "`",
			Want:       "\"`\"",
			WantLabels: "Literal",
		},
		{
			Input:      `'`,
			Want:       `"'"`,
			WantLabels: "Literal",
		},
		{
			Input:      "file:a",
			Want:       `"file:a"`,
			WantLabels: "None",
		},
		{
			Input:      `"file:a"`,
			Want:       `"\"file:a\""`,
			WantLabels: "Literal",
		},
		{
			Input:      `"x foo:bar`,
			Want:       `(concat "\"x" "foo:bar")`,
			WantLabels: "Literal",
		},
		// -repo:c" is considered valid. "repo:b is a literal pattern.
		{
			Input:      `"repo:b -repo:c"`,
			Want:       `(and "-repo:c\"" "\"repo:b")`,
			WantLabels: "Literal",
		},
		{
			Input:      `".*"`,
			Want:       `"\".*\""`,
			WantLabels: "Literal",
		},
		{
			Input:      `-pattern: ok`,
			Want:       `(concat "-pattern:" "ok")`,
			WantLabels: "Literal",
		},
		{
			Input:      `a:b "patterntype:regexp"`,
			Want:       `(concat "a:b" "\"patterntype:regexp\"")`,
			WantLabels: "Literal",
		},
		{
			Input:      `not file:foo pattern`,
			Want:       `(and "-file:foo" "pattern")`,
			WantLabels: "Literal",
		},
		{
			Input:      `not literal.*pattern`,
			Want:       `"NOT literal.*pattern"`,
			WantLabels: "Literal",
		},
		// Whitespace is removed. content: exists for preserving whitespace.
		{
			Input:      `lang:go func  main`,
			Want:       `(and "lang:go" (concat "func" "main"))`,
			WantLabels: "Literal",
		},
		{
			Input:      `\n`,
			Want:       `"\\n"`,
			WantLabels: "Literal",
		},
		{
			Input:      `\t`,
			Want:       `"\\t"`,
			WantLabels: "Literal",
		},
		{
			Input:      `\\`,
			Want:       `"\\\\"`,
			WantLabels: "Literal",
		},
		{
			Input:      `foo\d "bar*"`,
			Want:       `(concat "foo\\d" "\"bar*\"")`,
			WantLabels: "Literal",
		},
		{
			Input:      `\d`,
			Want:       `"\\d"`,
			WantLabels: "Literal",
		},
		{
			Input:      `type:commit message:"a commit message" after:"10 days ago"`,
			Want:       `(and "type:commit" "message:a commit message" "after:10 days ago")`,
			WantLabels: "None",
		},
		{
			Input:      `type:commit message:"a commit message" after:"10 days ago" test test2`,
			Want:       `(and "type:commit" "message:a commit message" "after:10 days ago" (concat "test" "test2"))`,
			WantLabels: "Literal",
		},
		{
			Input:      `type:commit message:'a commit message' after:'10 days ago' test test2`,
			Want:       `(and "type:commit" "message:a commit message" "after:10 days ago" (concat "test" "test2"))`,
			WantLabels: "Literal",
		},
		{
			Input:      `type:commit message:"a com"mit message" after:"10 days ago"`,
			Want:       `(and "type:commit" "message:a com" "after:10 days ago" (concat "mit" "message\""))`,
			WantLabels: "Literal",
		},
		{
			Input:      `bar and (foo or x\) ()`,
			Want:       `(or (and "bar" "(foo") (concat "x\\)" "()"))`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,HeuristicParensAsPatterns,Literal",
		},
		// For implementation simplicity, behavior preserves whitespace
		// inside parentheses.
		{
			Input:      "repo:foo (lisp    lisp)",
			Want:       `(and "repo:foo" "(lisp    lisp)")`,
			WantLabels: "HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      "repo:foo main( or (lisp    lisp)",
			Want:       `(and "repo:foo" (or "main(" "(lisp    lisp)"))`,
			WantLabels: "HeuristicHoisted,HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      "repo:foo )main( or (lisp    lisp)",
			Want:       `(and "repo:foo" (or ")main(" "(lisp    lisp)"))`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      "repo:foo ) main( or (lisp    lisp)",
			Want:       `(and "repo:foo" (or (concat ")" "main(") "(lisp    lisp)"))`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,HeuristicParensAsPatterns,Literal",
		},
		{
			Input:      "repo:foo )))) main( or (lisp    lisp) and )))",
			Want:       `(and "repo:foo" (or (concat "))))" "main(") (and "(lisp    lisp)" ")))")))`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,HeuristicParensAsPatterns,Literal",
		},

		{
			Input:      `"quoted"`,
			Want:       `"\"quoted\""`,
			WantLabels: "Literal",
		},
		{
			Input:      `repo:foo Args or main)`,
			Want:       `(and "repo:foo" (or "Args" "main)"))`,
			WantLabels: "HeuristicDanglingParens,HeuristicHoisted,Literal",
		},
		{
			Input:      `repo:foo Args) and main`,
			Want:       `(and "repo:foo" "Args)" "main")`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      `repo:foo bar and baz)`,
			Want:       `(and "repo:foo" "bar" "baz)")`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      `repo:foo bar)) and baz`,
			Want:       `(and "repo:foo" "bar))" "baz")`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      `repo:foo (bar( and baz())`,
			Want:       `(and "repo:foo" "bar(" "baz()")`,
			WantLabels: "Literal",
		},
		{
			Input:      `repo:foo (bar and baz))`,
			WantError:  `i'm having trouble understanding that query. The combination of parentheses is the problem. Try using the content: filter to quote patterns that contain parentheses`,
			WantLabels: "None",
		},
		{
			Input:      `repo:foo (bar and (baz)))`,
			WantError:  `i'm having trouble understanding that query. The combination of parentheses is the problem. Try using the content: filter to quote patterns that contain parentheses`,
			WantLabels: "None",
		},
		// This test input should error because the single quote in 'after' is unclosed.
		{
			Input:      `type:commit message:'a commit message' after:'10 days ago" test test2`,
			WantError:  "unterminated literal: expected '",
			WantLabels: "None",
		},
		// Fringe tests cases at the boundary of heuristics and invalid syntax.
		{
			Input:      `)(0 )0`,
			Want:       `(concat ")(0" ")0")`,
			WantLabels: "HeuristicDanglingParens,Literal",
		},
		{
			Input:      `((R:)0))0`,
			WantError:  `invalid query syntax`,
			WantLabels: "None",
		},
	}
	for _, tt := range cases {
		t.Run("literal search parse", func(t *testing.T) {
			result, err := ParseAndOr(tt.Input, SearchTypeLiteral)
			if err != nil {
				if diff := cmp.Diff(tt.WantError, err.Error()); diff != "" {
					t.Error(diff)
				}
			}
			var resultStr []string
			for _, node := range result {
				resultStr = append(resultStr, node.String())
			}
			got := strings.Join(resultStr, " ")
			if diff := cmp.Diff(tt.Want, got); diff != "" {
				t.Error(diff)
			}
			gotLabels := heuristicLabels(result)
			if diff := cmp.Diff(tt.WantLabels, gotLabels); diff != "" {
				t.Error(diff)
			}
		})
	}
}
