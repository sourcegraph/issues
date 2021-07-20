package graphqlbackend

import (
	"strconv"
	"sync/atomic"

	"github.com/cockroachdb/errors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/visitor"
	"github.com/inconshreveable/log15"
	"github.com/throttled/throttled/v2"

	"github.com/sourcegraph/sourcegraph/internal/conf"
	"github.com/sourcegraph/sourcegraph/internal/trace"
	"github.com/sourcegraph/sourcegraph/schema"
)

// Included in tracing so that we can differentiate different costs as we tweak
// the algorithm
const costEstimateVersion = 2

type QueryCost struct {
	FieldCount int
	MaxDepth   int
	Version    int
}

// EstimateQueryCost estimates the cost of the query before it is actually
// executed. It is a worst cast estimate of the number of fields expected to be
// returned by the query and handles nested queries a well as fragments.
func EstimateQueryCost(query string, variables map[string]interface{}) (totalCost *QueryCost, err error) {
	// NOTE: When we encounter errors in our visit funcs we return
	// visitor.ActionBreak to stop walking the tree and set the top level err
	// variable so that it is returned
	totalCost = &QueryCost{
		Version: costEstimateVersion,
	}

	// TODO: Remove this. It's here as a safeguard until we've run over a large
	// number of real world queries.
	defer func() {
		if r := recover(); r != nil {
			totalCost = nil
			err = r.(error)
		}
	}()
	if variables == nil {
		variables = make(map[string]interface{})
	}

	doc, err := parser.Parse(parser.ParseParams{
		Source: query,
	})
	if err != nil {
		return nil, errors.Wrap(err, "parsing query")
	}

	// We need to separate operations from fragments
	var operations []ast.Node
	var fragments []*ast.FragmentDefinition

	for i, def := range doc.Definitions {
		switch def.GetKind() {
		case kinds.FragmentDefinition:
			frag, ok := doc.Definitions[i].(*ast.FragmentDefinition)
			if !ok {
				return nil, errors.Errorf("expected FragmentDefinition, got %T", doc.Definitions[i])
			}
			fragments = append(fragments, frag)
		case kinds.OperationDefinition:
			operations = append(operations, doc.Definitions[i])
		}
	}

	// Calculate fragment costs first as we'll need them for the overall operation
	// cost.
	fragmentCosts := make(map[string]int)
	// Fragments can reference other fragments so we need their dependencies.
	fragmentDeps := make(map[string]map[string]struct{})

	for _, frag := range fragments {
		deps := getFragmentDependencies(frag)
		fragmentDeps[frag.Name.Value] = deps
	}

	// Checks whether we already have all the costs associated
	// with fragments included in the fragment fragName
	haveDepCosts := func(fragName string) bool {
		deps := fragmentDeps[fragName]
		for dep := range deps {
			_, ok := fragmentCosts[dep]
			if !ok {
				return false
			}
		}
		return true
	}

	fragSeen := make(map[string]struct{})

	for {
		for _, frag := range fragments {
			// Only try and calculate fragment cost if we've seen
			// all fragments it depends on.
			if !haveDepCosts(frag.Name.Value) {
				continue
			}
			cost, err := calcNodeCost(frag, fragmentCosts, variables)
			if err != nil {
				return nil, errors.Wrap(err, "calculating fragment cost")
			}
			fragmentCosts[frag.Name.Value] = cost.FieldCount
			fragSeen[frag.Name.Value] = struct{}{}
		}
		if len(fragSeen) == len(fragments) {
			break
		}
	}

	for _, def := range operations {
		cost, err := calcNodeCost(def, fragmentCosts, variables)
		if err != nil {
			return nil, errors.Wrap(err, "calculating operation cost")
		}
		totalCost.FieldCount += cost.FieldCount
		if totalCost.MaxDepth < cost.MaxDepth {
			totalCost.MaxDepth = cost.MaxDepth
		}
	}

	if totalCost.FieldCount < 1 {
		totalCost.FieldCount = 1
	}
	if totalCost.MaxDepth < 1 {
		totalCost.MaxDepth = 1
	}

	return totalCost, nil
}

func calcNodeCost(def ast.Node, fragmentCosts map[string]int, variables map[string]interface{}) (*QueryCost, error) {
	// NOTE: When we encounter errors in our visit funcs we return
	// visitor.ActionBreak to stop walking the tree and set the top level err
	// variable so that it is returned
	var visitErr error

	if fragmentCosts == nil {
		fragmentCosts = make(map[string]int)
	}
	inlineFragmentDepth := 0
	var inlineFragments []string

	// limitStack keeps track of the limit as we increase and decrease our depth in
	// the tree and encounter limit values
	limitStack := make([]int, 0)
	currentLimit := 1

	fieldCount := 0
	depth := 0
	maxDepth := 0
	multiplier := 1

	pushLimit := func() {
		multiplier = multiplier * currentLimit
		limitStack = append(limitStack, currentLimit)
		// Set limit back to 1 as we've already used it to increase our multiplier
		currentLimit = 1
	}
	popLimit := func() {
		if len(limitStack) == 0 {
			return
		}
		currentLimit = limitStack[len(limitStack)-1]
		limitStack = limitStack[:len(limitStack)-1]
		multiplier = multiplier / currentLimit
	}

	nonNullVariables := make(map[string]interface{})
	defaultValues := make(map[string]interface{})

	v := &visitor.VisitorOptions{
		Enter: func(p visitor.VisitFuncParams) (string, interface{}) {
			switch node := p.Node.(type) {
			case *ast.SelectionSet:
				depth++
				if depth > maxDepth {
					maxDepth = depth
				}
				pushLimit()
			case *ast.Field:
				switch node.Name.Value {
				// Values that won't appear in the result
				case "nodes", "__typename":
					return visitor.ActionNoChange, nil
				}
				if inlineFragmentDepth > 0 {
					// We don't count fields inside of inline fragments as we need to count all fragments
					// first to pick the largest one
					return visitor.ActionNoChange, nil
				}
				fieldCount += multiplier
			case *ast.VariableDefinition:
				// Track which variables are nonNull.
				if _, nonNull := node.Type.(*ast.NonNull); nonNull {
					nonNullVariables[node.Variable.Name.Value] = struct{}{}
				}
				if node.DefaultValue == nil {
					return visitor.ActionNoChange, nil
				}
				// Record default values
				switch v := node.DefaultValue.(type) {
				case *ast.IntValue:
					// Yes, IntValue's value is a string...
					defaultValues[node.Variable.Name.Value] = v.Value
				}
			case *ast.Variable:
				// We may have a limit variable
				if !shouldCheckParam(p) {
					return visitor.ActionNoChange, nil
				}
				limitVar, ok := variables[node.Name.Value]
				if !ok {
					if _, nonNull := nonNullVariables[node.Name.Value]; nonNull {
						visitErr = errors.Errorf("missing nonnull variable: %q", node.Name.Value)
						return visitor.ActionBreak, nil
					}
					if v, ok := defaultValues[node.Name.Value]; ok {
						// Pick default value if it was defined
						limitVar = v
					} else {
						// Fall back to a default of 1
						currentLimit = 1
						return visitor.ActionNoChange, nil
					}
				}
				limit, err := extractInt(limitVar)
				if err != nil {
					visitErr = errors.Wrap(err, "extracting limit")
					return visitor.ActionBreak, nil
				}
				if limit <= 0 {
					return visitor.ActionNoChange, nil
				}
				currentLimit = limit
			case *ast.IntValue:
				// We may have a limit
				if !shouldCheckParam(p) {
					return visitor.ActionNoChange, nil
				}
				limit, err := strconv.Atoi(node.Value)
				if err != nil {
					visitErr = errors.Wrap(err, "parsing limit")
					return visitor.ActionBreak, nil
				}
				if limit <= 0 {
					return visitor.ActionNoChange, nil
				}
				currentLimit = limit
			case *ast.FragmentSpread:
				fragmentCost, ok := fragmentCosts[node.Name.Value]
				if !ok {
					visitErr = errors.Errorf("unknown fragment %q", node.Name.Value)
					return visitor.ActionBreak, nil
				}
				fieldCount += fragmentCost * multiplier
			case *ast.InlineFragment:
				inlineFragmentDepth++
				// We calculate inline fragment costs and store them
				var fragCost *QueryCost
				fragCost, err := calcNodeCost(node.SelectionSet, fragmentCosts, variables)
				if err != nil {
					visitErr = errors.Wrap(err, "calculating inline fragment cost")
					return visitor.ActionBreak, nil
				}
				fragmentCosts[node.TypeCondition.Name.Value] = fragCost.FieldCount * multiplier
				inlineFragments = append(inlineFragments, node.TypeCondition.Name.Value)
			}
			return visitor.ActionNoChange, nil
		},
		Leave: func(p visitor.VisitFuncParams) (string, interface{}) {
			switch p.Node.(type) {
			case *ast.SelectionSet:
				depth--
				popLimit()
			case *ast.InlineFragment:
				inlineFragmentDepth--
			}
			return visitor.ActionNoChange, nil
		},
	}

	_ = visitor.Visit(def, v, nil)

	// We also need to pick the largest inline fragment in our tree
	var maxInlineFragmentCost int
	for _, v := range inlineFragments {
		fragCost := fragmentCosts[v]
		if fragCost > maxInlineFragmentCost {
			maxInlineFragmentCost = fragCost
		}
	}

	return &QueryCost{
		FieldCount: fieldCount + maxInlineFragmentCost,
		MaxDepth:   maxDepth,
	}, visitErr
}

// getFragmentDependencies returns all the fragments this node depend on.
func getFragmentDependencies(node ast.Node) map[string]struct{} {
	deps := make(map[string]struct{})

	v := &visitor.VisitorOptions{
		Enter: func(p visitor.VisitFuncParams) (string, interface{}) {
			switch node := p.Node.(type) {
			case *ast.FragmentSpread:
				deps[node.Name.Value] = struct{}{}
			}
			return visitor.ActionNoChange, nil
		},
	}

	_ = visitor.Visit(node, v, nil)

	return deps
}

func extractInt(i interface{}) (int, error) {
	switch v := i.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	case nil:
		return 0, nil
	default:
		return 0, errors.Errorf("unknown limit type: %T", i)
	}
}

var quantityParams = map[string]struct{}{
	"first": {},
	"last":  {},
}

func shouldCheckParam(p visitor.VisitFuncParams) bool {
	parent, ok := p.Parent.(*ast.Argument)
	if !ok {
		return false
	}
	if parent.Name == nil {
		return false
	}
	if _, ok := quantityParams[parent.Name.Value]; !ok {
		return false
	}
	return true
}

type LimiterArgs struct {
	IsIP          bool
	Anonymous     bool
	RequestName   string
	RequestSource trace.SourceType
}

type Limiter interface {
	RateLimit(key string, quantity int, args LimiterArgs) (bool, throttled.RateLimitResult, error)
}

type LimitWatcher interface {
	Get() (Limiter, bool)
}

func NewBasicLimitWatcher(store throttled.GCRAStore) *BasicLimitWatcher {
	basic := &BasicLimitWatcher{
		store: store,
	}
	conf.Watch(func() {
		e := conf.Get().ExperimentalFeatures
		if e == nil {
			basic.updateFromConfig(0)
			return
		}
		basic.updateFromConfig(e.RateLimitAnonymous)
	})
	return basic
}

type BasicLimitWatcher RateLimitWatcher

func (bl *BasicLimitWatcher) updateFromConfig(limit int) {
	if limit <= 0 {
		bl.rl.Store(&BasicLimiter{nil, false})
		log15.Debug("BasicLimiter disabled")
		return
	}
	maxBurstPercentage := 0.2
	l, err := throttled.NewGCRARateLimiter(
		bl.store,
		throttled.RateQuota{
			MaxRate:  throttled.PerHour(limit),
			MaxBurst: int(float64(limit) * maxBurstPercentage)},
	)
	if err != nil {
		log15.Warn("error updating BasicLimiter from config")
		bl.rl.Store(&BasicLimiter{nil, false})
		return
	}
	bl.rl.Store(&BasicLimiter{l, true})
	log15.Debug("BasicLimiter: rate limit updated", "new limit", limit)
}

// Get returns the latest Limiter.
func (bl *BasicLimitWatcher) Get() (Limiter, bool) {
	if l, ok := bl.rl.Load().(*BasicLimiter); ok {
		return l, l.enabled
	}
	return nil, false
}

type BasicLimiter struct {
	*throttled.GCRARateLimiter
	enabled bool
}

// RateLimit limits unauthenticated requests to the GraphQL API with an equal
// quantity of 1.
func (bl *BasicLimiter) RateLimit(_ string, _ int, args LimiterArgs) (bool, throttled.RateLimitResult, error) {
	if args.Anonymous && args.RequestName == "unknown" && args.RequestSource == trace.SourceOther && bl.GCRARateLimiter != nil {
		return bl.GCRARateLimiter.RateLimit("basic", 1)
	}
	return false, throttled.RateLimitResult{}, nil
}

// RateLimitWatcher stores the currently configured rate limiter and whether or
// not rate limiting is enabled.
type RateLimitWatcher struct {
	store throttled.GCRAStore
	rl    atomic.Value // *RateLimiter
}

// NewRateLimiteWatcher creates a new limiter with the provided store and starts
// watching for config changes.
func NewRateLimiteWatcher(store throttled.GCRAStore) *RateLimitWatcher {
	w := &RateLimitWatcher{
		store: store,
	}

	conf.Watch(func() {
		log15.Debug("Rate limit config updated, applying changes")
		w.updateFromConfig(conf.Get().ApiRatelimit)
	})

	return w
}

// Get returns the current rate limiter. If rate limiting is currently disabled
// (nil, false) is returned.
func (w *RateLimitWatcher) Get() (Limiter, bool) {
	if l, ok := w.rl.Load().(*RateLimiter); ok && l.enabled {
		return l, true
	}
	return nil, false
}

func (w *RateLimitWatcher) updateFromConfig(rlc *schema.ApiRatelimit) {
	// We can burst up to a max of 20% of limit
	maxBurstPercentage := 0.2

	if rlc == nil || !rlc.Enabled {
		w.rl.Store(&RateLimiter{enabled: false})
		return
	}

	ipQuota := throttled.RateQuota{
		MaxRate:  throttled.PerHour(rlc.PerIP),
		MaxBurst: int(float64(rlc.PerIP) * maxBurstPercentage),
	}
	ipLimiter, err := throttled.NewGCRARateLimiter(w.store, ipQuota)
	if err != nil {
		log15.Warn("error creating ip rate limiter", "error", err)
		return
	}

	userQuota := throttled.RateQuota{
		MaxRate:  throttled.PerHour(rlc.PerUser),
		MaxBurst: int(float64(rlc.PerUser) * maxBurstPercentage),
	}
	userLimiter, err := throttled.NewGCRARateLimiter(w.store, userQuota)
	if err != nil {
		log15.Warn("error creating user rate limiter", "error", err)
		return
	}

	overrides := make(map[string]limiter)
	for _, o := range rlc.Overrides {
		switch l := o.Limit.(type) {
		case string:
			if l == "blocked" {
				overrides[o.Key] = &fixedLimiter{
					limited: true,
					result: throttled.RateLimitResult{
						Limit:      0,
						Remaining:  0,
						ResetAfter: 0,
						RetryAfter: 0,
					},
				}
			} else if l == "unlimited" {
				overrides[o.Key] = &fixedLimiter{
					limited: false,
					result: throttled.RateLimitResult{
						Limit:      100000,
						Remaining:  100000,
						ResetAfter: 0,
						RetryAfter: 0,
					},
				}
			} else {
				log15.Warn("unknown limit value", "value", l)
				return
			}
		case int:
			rl, err := throttled.NewGCRARateLimiter(w.store, throttled.RateQuota{
				MaxRate:  throttled.PerHour(l),
				MaxBurst: int(float64(l) * maxBurstPercentage),
			})
			if err != nil {
				log15.Warn("error creating override rate limiter", "key", o.Key, "error", err)
				return
			}
			overrides[o.Key] = rl
		}
	}

	// Store the new limiter
	w.rl.Store(&RateLimiter{
		enabled:     true,
		ipLimiter:   ipLimiter,
		userLimiter: userLimiter,
		overrides:   overrides,
	})
}

type RateLimiter struct {
	enabled     bool
	ipLimiter   *throttled.GCRARateLimiter
	userLimiter *throttled.GCRARateLimiter
	overrides   map[string]limiter
}

func (rl *RateLimiter) RateLimit(uid string, cost int, args LimiterArgs) (bool, throttled.RateLimitResult, error) {
	if r, ok := rl.overrides[uid]; ok {
		return r.RateLimit(uid, cost)
	}
	if args.IsIP {
		return rl.ipLimiter.RateLimit(uid, cost)
	}
	return rl.userLimiter.RateLimit(uid, cost)
}

type limiter interface {
	RateLimit(string, int) (bool, throttled.RateLimitResult, error)
}

// fixedLimiter is a rate limiter that always returns the same result
type fixedLimiter struct {
	limited bool
	result  throttled.RateLimitResult
}

func (f *fixedLimiter) RateLimit(string, int) (bool, throttled.RateLimitResult, error) {
	return f.limited, f.result, nil
}
