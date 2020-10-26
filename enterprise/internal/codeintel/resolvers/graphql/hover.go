package graphql

import (
	"github.com/sourcegraph/go-lsp"
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
)

type HoverResolver struct {
	text     string
	lspRange lsp.Range
}

func NewHoverResolver(text string, lspRange lsp.Range) gql.HoverResolver {
	return &HoverResolver{
		text:     text,
		lspRange: lspRange,
	}
}

func (r *HoverResolver) Markdown() gql.Markdown   { return gql.Markdown(r.text) }
func (r *HoverResolver) Range() gql.RangeResolver { return gql.NewRangeResolver(r.lspRange) }
