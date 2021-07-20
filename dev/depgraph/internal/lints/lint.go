package lints

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"

	"github.com/sourcegraph/sourcegraph/dev/depgraph/internal/graph"
)

type Lint func(graph *graph.DependencyGraph) []lintError

type lintError struct {
	pkg     string
	message []string
}

var lintsByName = map[string]Lint{
	"NoBinarySpecificSharedCode": NoBinarySpecificSharedCode,
	"NoDeadPackages":             NoDeadPackages,
	"NoEnterpriseImportsFromOSS": NoEnterpriseImportsFromOSS,
	"NoLooseCommands":            NoLooseCommands,
	"NoReachingIntoCommands":     NoReachingIntoCommands,
	"NoUnusedSharedCommandCode":  NoUnusedSharedCommandCode,
}

var DefaultLints []string

func init() {
	for name := range lintsByName {
		DefaultLints = append(DefaultLints, name)
	}
}

// Run runs the lint passes with the given names using the given graph. The lint
// violations will be formatted as a non-nil error value.
func Run(graph *graph.DependencyGraph, names []string) error {
	lints := make([]Lint, 0, len(names))
	for _, name := range names {
		lint, ok := lintsByName[name]
		if !ok {
			return errors.Errorf("unknown lint '%s'", name)
		}

		lints = append(lints, lint)
	}

	var errs []lintError
	for _, lint := range lints {
		errs = append(errs, lint(graph)...)
	}

	return formatErrors(errs)
}

// maxNumErrors is the maxmum number of errors that will be displayed at once.
const maxNumErrors = 500

// formatErrors returns an error value that is formatted to display the given lint
// errors. If there were no lint errors, this function will return nil.
func formatErrors(errs []lintError) error {
	if len(errs) == 0 {
		return nil
	}

	sort.Slice(errs, func(i, j int) bool {
		return errs[i].pkg < errs[j].pkg || (errs[i].pkg == errs[j].pkg && strings.Join(errs[i].message, "\n") < strings.Join(errs[j].message, "\n"))
	})

	preamble := fmt.Sprintf("%d lint violations", len(errs))

	if len(errs) > maxNumErrors {
		errs = errs[:maxNumErrors]
		preamble += fmt.Sprintf(" (showing %d)", len(errs))
	}

	items := make([]string, 0, len(errs))
	for i, err := range errs {
		pkg := err.pkg
		if pkg == "" {
			pkg = "<root>"
		}

		items = append(items, fmt.Sprintf("%3d. %s\n     %s\n", i+1, pkg, strings.Join(err.message, "\n     ")))
	}

	return errors.Errorf("%s:\n\n%s", preamble, strings.Join(items, "\n"))
}
