package query

// The Visitor interface allows to visit nodes for each respective part of the
// query grammar.
type Visitor interface {
	VisitNodes(v Visitor, node []Node)
	VisitOperator(v Visitor, kind operatorKind, operands []Node)
	VisitParameter(v Visitor, field, value string, negated bool, annotation Annotation)
	VisitPattern(v Visitor, value string, negated bool, annotation Annotation)
}

// BaseVisitor is a visitor that recursively visits each node in a query. A
// BaseVisitor's methods may be overriden by embedding it a custom visitor's
// definition. See OperatorVisitor for an example.
type BaseVisitor struct{}

func (*BaseVisitor) VisitNodes(visitor Visitor, nodes []Node) {
	for _, node := range nodes {
		switch v := node.(type) {
		case Pattern:
			visitor.VisitPattern(visitor, v.Value, v.Negated, v.Annotation)
		case Parameter:
			visitor.VisitParameter(visitor, v.Field, v.Value, v.Negated, v.Annotation)
		case Operator:
			visitor.VisitOperator(visitor, v.Kind, v.Operands)
		default:
			panic("unreachable")
		}
	}
}

func (*BaseVisitor) VisitOperator(visitor Visitor, kind operatorKind, operands []Node) {
	visitor.VisitNodes(visitor, operands)
}

func (*BaseVisitor) VisitParameter(visitor Visitor, field, value string, negated bool, annotation Annotation) {
}

func (*BaseVisitor) VisitPattern(visitor Visitor, value string, negated bool, annotation Annotation) {
}

// ParameterVisitor is a helper visitor that only visits operators in a query,
// and supplies the operator members via a callback.
type OperatorVisitor struct {
	BaseVisitor
	callback func(kind operatorKind, operands []Node)
}

func (s *OperatorVisitor) VisitOperator(visitor Visitor, kind operatorKind, operands []Node) {
	s.callback(kind, operands)
	visitor.VisitNodes(visitor, operands)
}

// ParameterVisitor is a helper visitor that only visits parameters in a query,
// and supplies the parameter members via a callback.
type ParameterVisitor struct {
	BaseVisitor
	callback func(field, value string, negated bool, annotation Annotation)
}

func (s *ParameterVisitor) VisitParameter(visitor Visitor, field, value string, negated bool, annotation Annotation) {
	s.callback(field, value, negated, annotation)
}

// PatternVisitor is a helper visitor that only visits patterns in a query,
// and supplies the pattern members via a callback.
type PatternVisitor struct {
	BaseVisitor
	callback func(value string, negated bool, annotation Annotation)
}

func (s *PatternVisitor) VisitPattern(visitor Visitor, value string, negated bool, annotation Annotation) {
	s.callback(value, negated, annotation)
}

// FieldVisitor is a helper visitor that only visits parameter fields in a
// query, for a field specified in the state. For each parameter with
// this field name it calls the callback with the field's members.
type FieldVisitor struct {
	BaseVisitor
	field    string
	callback func(value string, negated bool, annotation Annotation)
}

func (s *FieldVisitor) VisitParameter(visitor Visitor, field, value string, negated bool, annotation Annotation) {
	if s.field == field {
		s.callback(value, negated, annotation)
	}
}

// VisitOperator is a convenience function that calls callback on all operator
// nodes. callback supplies the node's kind and operands.
func VisitOperator(nodes []Node, callback func(kind operatorKind, operands []Node)) {
	visitor := &OperatorVisitor{callback: callback}
	visitor.VisitNodes(visitor, nodes)
}

// VisitParameter is a convenience function that calls callback on all parameter
// nodes. callback supplies the node's field, value, and whether the value is
// negated.
func VisitParameter(nodes []Node, callback func(field, value string, negated bool, annotation Annotation)) {
	visitor := &ParameterVisitor{callback: callback}
	visitor.VisitNodes(visitor, nodes)
}

// VisitPattern is a convenience function that calls callback on all pattern
// nodes. callback supplies the node's value value, and whether the value is
// negated or quoted.
func VisitPattern(nodes []Node, callback func(value string, negated bool, annotation Annotation)) {
	visitor := &PatternVisitor{callback: callback}
	visitor.VisitNodes(visitor, nodes)
}

// VisitField convenience function that calls callback on all parameter nodes
// whose field matches the field argument. callback supplies the node's value
// and whether the value is negated.
func VisitField(nodes []Node, field string, callback func(value string, negated bool, annotation Annotation)) {
	visitor := &FieldVisitor{callback: callback, field: field}
	visitor.VisitNodes(visitor, nodes)
}
