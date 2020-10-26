package lsif

type Element struct {
	ID      int
	Type    string
	Label   string
	Payload interface{}
}

type Edge struct {
	OutV     int
	InV      int
	InVs     []int
	Document int
}

type MetaData struct {
	Version     string
	ProjectRoot string
}

type Range struct {
	StartLine          int
	StartCharacter     int
	EndLine            int
	EndCharacter       int
	DefinitionResultID int
	ReferenceResultID  int
	HoverResultID      int
}

func (d Range) SetDefinitionResultID(id int) Range {
	return Range{
		StartLine:          d.StartLine,
		StartCharacter:     d.StartCharacter,
		EndLine:            d.EndLine,
		EndCharacter:       d.EndCharacter,
		DefinitionResultID: id,
		ReferenceResultID:  d.ReferenceResultID,
		HoverResultID:      d.HoverResultID,
	}
}

func (d Range) SetReferenceResultID(id int) Range {
	return Range{
		StartLine:          d.StartLine,
		StartCharacter:     d.StartCharacter,
		EndLine:            d.EndLine,
		EndCharacter:       d.EndCharacter,
		DefinitionResultID: d.DefinitionResultID,
		ReferenceResultID:  id,
		HoverResultID:      d.HoverResultID,
	}
}

func (d Range) SetHoverResultID(id int) Range {
	return Range{
		StartLine:          d.StartLine,
		StartCharacter:     d.StartCharacter,
		EndLine:            d.EndLine,
		EndCharacter:       d.EndCharacter,
		DefinitionResultID: d.DefinitionResultID,
		ReferenceResultID:  d.ReferenceResultID,
		HoverResultID:      id,
	}
}

type ResultSet struct {
	DefinitionResultID int
	ReferenceResultID  int
	HoverResultID      int
}

func (d ResultSet) SetDefinitionResultID(id int) ResultSet {
	return ResultSet{
		DefinitionResultID: id,
		ReferenceResultID:  d.ReferenceResultID,
		HoverResultID:      d.HoverResultID,
	}
}

func (d ResultSet) SetReferenceResultID(id int) ResultSet {
	return ResultSet{
		DefinitionResultID: d.DefinitionResultID,
		ReferenceResultID:  id,
		HoverResultID:      d.HoverResultID,
	}
}

func (d ResultSet) SetHoverResultID(id int) ResultSet {
	return ResultSet{
		DefinitionResultID: d.DefinitionResultID,
		ReferenceResultID:  d.ReferenceResultID,
		HoverResultID:      id,
	}
}

type Moniker struct {
	Kind                 string
	Scheme               string
	Identifier           string
	PackageInformationID int
}

func (d Moniker) SetPackageInformationID(id int) Moniker {
	return Moniker{
		Kind:                 d.Kind,
		Scheme:               d.Scheme,
		Identifier:           d.Identifier,
		PackageInformationID: id,
	}
}

type PackageInformation struct {
	Name    string
	Version string
}

type Diagnostic struct {
	Severity       int
	Code           string
	Message        string
	Source         string
	StartLine      int
	StartCharacter int
	EndLine        int
	EndCharacter   int
}
