package priority

// Cost is an approximation of the resource demand of a Sourcegraph query, or a set of queries. This is useful
// for insight query execution strategies.
type Cost int

const (
	Indexed   Cost = 500
	Unindexed Cost = 5000 // using an order of magnitude approximation at the moment. Eventually this should become something a little smarter.
)
