// Code generated by "stringer -type=TokenType"; DO NOT EDIT.

package token

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Unspecified-0]
	_ = x[NilLiteral-1]
	_ = x[BoolLiteral-2]
	_ = x[IntLiteral-3]
	_ = x[DoubleLiteral-4]
	_ = x[StrLiteral-5]
	_ = x[Plus-6]
	_ = x[Minus-7]
	_ = x[Times-8]
	_ = x[Div-9]
	_ = x[Modulo-10]
	_ = x[Equal-11]
	_ = x[NotEqual-12]
	_ = x[LessThan-13]
	_ = x[LessEq-14]
	_ = x[GreaterThan-15]
	_ = x[GreaterEq-16]
	_ = x[LogicalAnd-17]
	_ = x[LogicalOr-18]
	_ = x[BitwiseOr-19]
	_ = x[BitwiseAnd-20]
	_ = x[BitwiseXor-21]
	_ = x[ShiftLeft-22]
	_ = x[ShiftRight-23]
	_ = x[Dot-24]
	_ = x[Not-25]
	_ = x[Complement-26]
	_ = x[Comma-27]
	_ = x[LeftParen-28]
	_ = x[RightParen-29]
	_ = x[LeftBrace-30]
	_ = x[RightBrace-31]
	_ = x[LeftSquare-32]
	_ = x[RightSquare-33]
	_ = x[Colon-34]
	_ = x[Ident-35]
	_ = x[Func-36]
	_ = x[Let-37]
	_ = x[Template-38]
	_ = x[If-39]
	_ = x[Then-40]
	_ = x[Else-41]
	_ = x[EndOfInput-42]
}

const _TokenType_name = "UnspecifiedNilLiteralBoolLiteralIntLiteralDoubleLiteralStrLiteralPlusMinusTimesDivModuloEqualNotEqualLessThanLessEqGreaterThanGreaterEqLogicalAndLogicalOrBitwiseOrBitwiseAndBitwiseXorShiftLeftShiftRightDotNotComplementCommaLeftParenRightParenLeftBraceRightBraceLeftSquareRightSquareColonIdentFuncLetTemplateIfThenElseEndOfInput"

var _TokenType_index = [...]uint16{0, 11, 21, 32, 42, 55, 65, 69, 74, 79, 82, 88, 93, 101, 109, 115, 126, 135, 145, 154, 163, 173, 183, 192, 202, 205, 208, 218, 223, 232, 242, 251, 261, 271, 282, 287, 292, 296, 299, 307, 309, 313, 317, 327}

func (i TokenType) String() string {
	if i < 0 || i >= TokenType(len(_TokenType_index)-1) {
		return "TokenType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TokenType_name[_TokenType_index[i]:_TokenType_index[i+1]]
}
