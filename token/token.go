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
	PlusOp
	MinusOp
	TimesOp
	DivOp
	Equal
	NotEqual
	LessThan
	LessEq
	GreaterThan
	GreaterEq
	Dot
	Not
	// Separators
	Comma
	LeftParen
	RightParen
	LeftBrace
	RightBrace
	Colon
	// Identifiers
	Ident
	Keyword
	// Don't treat end of input as an error, but use a special token.
	EndOfInput
)

type Token struct {
	Typ TokenType
	Pos int
	End int
	Val string
}
