// Code generated by "stringer -type=TokenType"; DO NOT EDIT.

package syntax

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TokenEOF-0]
	_ = x[TokenError-1]
	_ = x[TokenLiteral-2]
	_ = x[TokenQuoted-3]
	_ = x[TokenPattern-4]
	_ = x[TokenColon-5]
	_ = x[TokenMinus-6]
	_ = x[TokenSep-7]
}

const _TokenType_name = "TokenEOFTokenErrorTokenLiteralTokenQuotedTokenPatternTokenColonTokenMinusTokenSep"

var _TokenType_index = [...]uint8{0, 8, 18, 30, 41, 53, 63, 73, 81}

func (i TokenType) String() string {
	if i < 0 || i >= TokenType(len(_TokenType_index)-1) {
		return "TokenType(" + strconv.Itoa(int64(i)) + ")"
	}
	return _TokenType_name[_TokenType_index[i]:_TokenType_index[i+1]]
}
