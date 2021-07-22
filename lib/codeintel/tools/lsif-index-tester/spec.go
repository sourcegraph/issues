package main

type Position struct {
	// zero-based line index
	Line int `json:"line"`

	// one-based character index
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type DefinitionRequest struct {
	TextDocument string   `json:"textDocument"`
	Position     Position `json:"position"`
}

type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

type DefinitionTest struct {
	Name     string            `json:"name"`
	Request  DefinitionRequest `json:"request"`
	Response Location          `json:"response"`
}

type ReferenceContext struct {
	IncludeDeclaration bool
}

type ReferenceRequest struct {
	TextDocument string           `json:"textDocument"`
	Position     Position         `json:"position"`
	Context      ReferenceContext `json:"context"`
}

type ReferenceResponse []Location

type ReferencesTest struct {
	Name     string            `json:"name"`
	Request  ReferenceRequest  `json:"request"`
	Response ReferenceResponse `json:"response"`
}

type LsifTest struct {
	Definitions []DefinitionTest `json:"textDocument/definition"`
	References  []ReferencesTest `json:"textDocument/references"`
}
