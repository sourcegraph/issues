package query

// The Mapper interface allows to replace nodes for each respective part of the
// query grammar. It is a visitor that will replace the visited node by the
// returned value.
type Mapper interface {
	MapNodes(m Mapper, node []Node) []Node
	MapOperator(m Mapper, kind operatorKind, operands []Node) []Node
	MapParameter(m Mapper, field, value string, negated bool, annotation Annotation) Node
	MapPattern(m Mapper, value string, negated bool, annotation Annotation) Node
}

// The BaseMapper is a mapper that recursively visits each node in a query and
// maps it to itself. A BaseMapper's methods may be overriden by embedding it a
// custom mapper's definition. See ParameterMapper for an example. If the
// methods return nil, the respective node is removed.
type BaseMapper struct{}

func (*BaseMapper) MapNodes(mapper Mapper, nodes []Node) []Node {
	mapped := []Node{}
	for _, node := range nodes {
		switch v := node.(type) {
		case Pattern:
			if result := mapper.MapPattern(mapper, v.Value, v.Negated, v.Annotation); result != nil {
				mapped = append(mapped, result)
			}
		case Parameter:
			if result := mapper.MapParameter(mapper, v.Field, v.Value, v.Negated, v.Annotation); result != nil {
				mapped = append(mapped, result)
			}
		case Operator:
			if result := mapper.MapOperator(mapper, v.Kind, v.Operands); result != nil {
				mapped = append(mapped, result...)
			}
		}
	}
	return mapped
}

// Base mapper for Operators. Reduces operands if changed.
func (*BaseMapper) MapOperator(mapper Mapper, kind operatorKind, operands []Node) []Node {
	return newOperator(mapper.MapNodes(mapper, operands), kind)
}

// Base mapper for Parameters. It is the identity function.
func (*BaseMapper) MapParameter(mapper Mapper, field, value string, negated bool, annotation Annotation) Node {
	return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
}

// Base mapper for Patterns. It is the identity function.
func (*BaseMapper) MapPattern(mapper Mapper, value string, negated bool, annotation Annotation) Node {
	return Pattern{Value: value, Negated: negated, Annotation: annotation}
}

// OperatorMapper is a helper mapper that maps operators in a query. It takes as
// state a callback that will call and map each visited operator with the return
// value.
type OperatorMapper struct {
	BaseMapper
	callback func(kind operatorKind, operands []Node) []Node
}

// MapOperator implements OperatorMapper by overriding the BaseMapper's value to
// substitute a node computed by the callback. It reduces any substituted node.
func (s *OperatorMapper) MapOperator(mapper Mapper, kind operatorKind, operands []Node) []Node {
	return newOperator(s.callback(kind, operands), And)
}

// ParameterMapper is a helper mapper that only maps parameters in a query. It
// takes as state a callback that will call and map each visited parameter by
// the return value.
type ParameterMapper struct {
	BaseMapper
	callback func(field, value string, negated bool, annotation Annotation) Node
}

// MapParameter implements ParameterMapper by overriding the BaseMapper's value
// to substitute a node computed by the callback.
func (s *ParameterMapper) MapParameter(mapper Mapper, field, value string, negated bool, annotation Annotation) Node {
	return s.callback(field, value, negated, annotation)
}

// PatternMapper is a helper mapper that only maps patterns in a query. It
// takes as state a callback that will call and map each visited pattern by
// the return value.
type PatternMapper struct {
	BaseMapper
	callback func(value string, negated bool, annotation Annotation) Node
}

func (s *PatternMapper) MapPattern(mapper Mapper, value string, negated bool, annotation Annotation) Node {
	return s.callback(value, negated, annotation)
}

// FieldMapper is a helper mapper that only maps patterns in a query, for a
// field specified in state. For each parameter with this field name it calls
// the callback that maps the field's members.
type FieldMapper struct {
	BaseMapper
	field    string
	callback func(value string, negated bool) Node
}

func (s *FieldMapper) MapParameter(mapper Mapper, field, value string, negated bool, annotation Annotation) Node {
	if s.field == field {
		return s.callback(value, negated)
	}
	return Parameter{Field: field, Value: value, Negated: negated, Annotation: annotation}
}

// MapOperator is a convenience function that calls callback on all operator
// nodes, substituting them for callback's return value. callback supplies the
// node's kind and operands.
func MapOperator(nodes []Node, callback func(kind operatorKind, operands []Node) []Node) []Node {
	mapper := &OperatorMapper{callback: callback}
	return mapper.MapNodes(mapper, nodes)
}

// MapParameter is a convenience function that calls callback on all parameter
// nodes, substituting them for callback's return value. callback supplies the
// node's field, value, and whether the value is negated.
func MapParameter(nodes []Node, callback func(field, value string, negated bool, annotation Annotation) Node) []Node {
	mapper := &ParameterMapper{callback: callback}
	return mapper.MapNodes(mapper, nodes)
}

// MapPattern is a convenience function that calls callback on all pattern
// nodes, substituting them for callback's return value. callback supplies the
// node's field, value, and whether the value is negated.
func MapPattern(nodes []Node, callback func(value string, negated bool, annotation Annotation) Node) []Node {
	mapper := &PatternMapper{callback: callback}
	return mapper.MapNodes(mapper, nodes)
}

// MapField is a convenience function that calls callback on all parameter nodes
// whose field matches the field argument, substituting them for callback's
// return value. callback supplies the node's value, and whether the value is
// negated.
func MapField(nodes []Node, field string, callback func(value string, negated bool) Node) []Node {
	mapper := &FieldMapper{callback: callback, field: field}
	return mapper.MapNodes(mapper, nodes)
}
