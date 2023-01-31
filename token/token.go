package token

//go:generate stringer -type=TokenType
type TokenType int32

const (
	Unspecified TokenType = iota
	// Literals
	NilLiteral
	BoolLiteral
	IntLiteral
	DoubleLiteral
	StrLiteral
	// Operators
	Plus
	Minus
	Times
	Div
	Modulo
	Equal
	NotEqual
	LessThan
	LessEq
	GreaterThan
	GreaterEq
	LogicalAnd
	LogicalOr
	BitwiseOr
	BitwiseAnd
	BitwiseXor
	ShiftLeft
	ShiftRight
	Dot
	Not
	Complement
	// Separators
	Comma
	LeftParen
	RightParen
	LeftBrace
	RightBrace
	Colon
	// Identifiers
	Ident
	// Keywords
	Func
	Let
	// Don't treat end of input as an error, but use a special token.
	EndOfInput
)

type Token struct {
	Typ TokenType
	Pos int
	End int
	Val string
}
