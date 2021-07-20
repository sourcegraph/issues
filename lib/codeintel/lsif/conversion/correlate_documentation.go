package conversion

import (
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/protocol"
	"github.com/sourcegraph/sourcegraph/lib/codeintel/lsif/protocol/reader"
)

// Correlation for the Sourcegraph API documentation extension to LSIF

func correlateDocumentationResult(state *wrappedState, element Element) error {
	payload, ok := element.Payload.(protocol.Documentation)
	if !ok {
		return ErrUnexpectedPayload
	}

	if payload.Tags == nil {
		// don't encode "null", instead encode an empty list. Null is forbidden for tags,
		// but it can crop up in some languages (e.g. Go) due to JSON encoders so we handle
		// it gracefully.
		payload.Tags = []protocol.Tag{}
	}
	state.DocumentationResultsData[element.ID] = payload
	return nil
}

func correlateDocumentationString(state *wrappedState, element Element) error {
	payload, ok := element.Payload.(protocol.MarkupContent)
	if !ok {
		return ErrUnexpectedPayload
	}

	state.DocumentationStringsData[element.ID] = payload
	return nil
}

func correlateDocumentationResultEdge(state *wrappedState, id int, edge Edge) error {
	documentationResult := edge.InV
	projectOrResultSet := edge.OutV

	if _, ok := state.DocumentationResultsData[documentationResult]; !ok {
		return malformedDump(id, documentationResult, "documentationResult")
	}

	if source, ok := state.ResultSetData[projectOrResultSet]; ok {
		state.ResultSetData[projectOrResultSet] = source.SetDocumentationResultID(documentationResult)
	} else {
		// the `project` vertices are not stored, but this condition indicates the root documentationResult
		// vertex was attached to the `project` vertex, and we want to store it.
		state.DocumentationResultRoot = documentationResult
	}
	return nil
}

func correlateDocumentationChildrenEdge(state *wrappedState, id int, edge Edge) error {
	children := edge.InVs
	parent := edge.OutV

	for _, child := range children {
		if _, ok := state.DocumentationResultsData[child]; !ok {
			return malformedDump(id, child, "documentationResult")
		}
	}
	if _, ok := state.DocumentationResultsData[parent]; !ok {
		return malformedDump(id, parent, "documentationResult")
	}
	state.DocumentationChildren[parent] = children
	return nil
}

func correlateDocumentationStringEdge(state *wrappedState, id int, edge reader.DocumentationStringEdge) error {
	documentationString := edge.InV
	documentationResult := edge.OutV

	if _, ok := state.DocumentationStringsData[documentationString]; !ok {
		return malformedDump(id, documentationString, "documentationString")
	}
	if _, ok := state.DocumentationResultsData[documentationResult]; !ok {
		return malformedDump(id, documentationResult, "documentationResult")
	}

	switch edge.Kind {
	case protocol.DocumentationStringKindLabel:
		state.DocumentationStringLabel[documentationResult] = documentationString
	case protocol.DocumentationStringKindDetail:
		state.DocumentationStringDetail[documentationResult] = documentationString
	default:
		panic("never here")
	}
	return nil
}
