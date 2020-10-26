package graphqlbackend

// A resolver for the GraphQL type GenericSearchMatch
type searchResultMatchResolver struct {
	url        string
	body       string
	highlights []*highlightedRange
}

func (m *searchResultMatchResolver) URL() string {
	return m.url
}

func (m *searchResultMatchResolver) Body() Markdown {
	return Markdown(m.body)
}

func (m *searchResultMatchResolver) Highlights() []*highlightedRange {
	return m.highlights
}
