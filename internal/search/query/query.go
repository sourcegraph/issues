package query

/*
Query processing involves multiple steps to produce a query to evaluate.

To unify multiple concerns, query processing is abstracted to a sequence of
steps that entail parsing, validity checking, transformation, and conditional
processing logic driven by external options.
*/

// A step performs a transformation on nodes, which may fail.
type step func(nodes []Node) ([]Node, error)

// A pass is a step that never fails.
type pass func(nodes []Node) []Node

// sequence sequences zero or more steps to create a single step.
func sequence(steps ...step) step {
	return func(nodes []Node) ([]Node, error) {
		var err error
		for _, step := range steps {
			nodes, err = step(nodes)
			if err != nil {
				return nil, err
			}
		}
		return nodes, nil
	}
}

// succeeds converts a sequence of passes into a single step.
func succeeds(passes ...pass) step {
	return func(nodes []Node) ([]Node, error) {
		for _, pass := range passes {
			nodes = pass(nodes)
		}
		return nodes, nil
	}
}

// With returns step if enabled is true. Use it to compose a pipeline that
// conditionally run steps.
func With(enabled bool, step step) step {
	if !enabled {
		return identity
	}
	return step
}

// For runs processing steps for a given search type. This includes
// normalization, substitution for whitespace, and pattern labeling.
func For(searchType SearchType) step {
	var processType step
	switch searchType {
	case SearchTypeLiteral:
		processType = succeeds(substituteConcat(space))
	case SearchTypeRegex:
		processType = succeeds(escapeParensHeuristic, substituteConcat(fuzzyRegexp))
	case SearchTypeStructural:
		processType = succeeds(labelStructural, ellipsesForHoles, substituteConcat(space))
	}
	normalize := succeeds(LowercaseFieldNames, SubstituteAliases(searchType), SubstituteCountAll)
	return sequence(normalize, processType)
}

// Init creates a step from an input string and search type. It parses the
// initial input string.
func Init(in string, searchType SearchType) step {
	parser := func([]Node) ([]Node, error) {
		return Parse(in, searchType)
	}
	return sequence(parser, For(searchType))
}

// InitLiteral is Init where SearchType is Literal.
func InitLiteral(in string) step {
	return Init(in, SearchTypeLiteral)
}

// InitRegexp is Init where SearchType is Regex.
func InitRegexp(in string) step {
	return Init(in, SearchTypeRegex)
}

// InitStructural is Init where SearchType is Structural.
func InitStructural(in string) step {
	return Init(in, SearchTypeStructural)
}

func Run(step step) ([]Node, error) {
	return step(nil)
}

func Validate(disjuncts [][]Node) error {
	for _, disjunct := range disjuncts {
		if err := validate(disjunct); err != nil {
			return err
		}
	}
	return nil
}

// A BasicPass is a transformation on Basic queries.
type BasicPass func(Basic) Basic

// MapPlan runs passes on all Basic queries in a plan. It expects a valid plan.
// It guarantees transformation succeeds.
func MapPlan(plan Plan, passes ...BasicPass) Plan {
	for _, pass := range passes {
		for i, query := range plan {
			plan[i] = pass(query)
		}
	}
	return plan
}

func ToPlan(disjuncts [][]Node) (Plan, error) {
	plan := make([]Basic, 0, len(disjuncts))
	for _, disjunct := range disjuncts {
		basic, err := ToBasicQuery(disjunct)
		if err != nil {
			return nil, err
		}
		plan = append(plan, basic)
	}
	return plan, nil
}

// Pipeline processes zero or more steps to produce a query. The first step must
// be Init, otherwise this function is a no-op.
func Pipeline(steps ...step) (Plan, error) {
	nodes, err := sequence(steps...)(nil)
	if err != nil {
		return nil, err
	}

	disjuncts := Dnf(nodes)
	if err := Validate(disjuncts); err != nil {
		return nil, err
	}

	plan, err := ToPlan(disjuncts)
	if err != nil {
		return nil, err
	}
	plan = MapPlan(plan, ConcatRevFilters, typeRepoToFilter)
	return plan, nil
}
