package token

//go:generate stringer -type=TokenType
type TokenType int32

const (
	Unspecified TokenType = iota
	// Literals
	Nil              // nil
	BoolLiteral      // true false
	IntLiteral       // 0 1 2
	DoubleLiteral    // 0. 1.2 3e-4
	StrLiteral       // "foo" 'bar'
	FormatStrLiteral // "/path/to/${heaven}"
	// Operators
	Plus        // +
	Minus       // -
	Times       // *
	Div         // /
	Modulo      // %
	Equal       // ==
	NotEqual    // !=
	LessThan    // <
	LessEq      // <=
	GreaterThan // >
	GreaterEq   // >=
	LogicalAnd  // &&
	LogicalOr   // ||
	BitwiseAnd  // &
	BitwiseOr   // |
	BitwiseXor  // ^
	ShiftLeft   // <<
	ShiftRight  // >>
	Dot         // .
	Not         // !
	Complement  // ~
	Merge       // @
	// Separators
	Comma       // ,
	LeftParen   // (
	RightParen  // )
	LeftBrace   // {
	RightBrace  // }
	LeftSquare  // [
	RightSquare // ]
	Colon       // :
	OfType      // ::
	// Identifiers
	Ident
	// Keywords
	Func     // func
	Let      // let
	Template // template
	If       // if
	Then     // then
	Else     // else
	// Don't treat end of input as an error, but use a special token.
	EndOfInput
)

type Token struct {
	Typ TokenType
	Pos Pos
	End Pos
	Val string
	Fmt *FormatStr
}

// Pos represents a (byte) offset into a File that is part of a FileSet.
// This representation is very similar to the one used in the Go compiler:
// https://cs.opensource.google/go/go/+/master:src/go/token/token.go
type Pos int

type Poser interface {
	Pos() Pos
}

// Types related to format strings.

type FormatStrPart struct {
	Val string
	Pos Pos
	End Pos
}
type FormattedValue struct {
	Tokens []Token
	Pos    Pos
	End    Pos
}

// FormatStrValue is a marker interface for types that can be part of a format string.
type FormatStrValue interface {
	formatStrValueImpl()
}

func (FormattedValue) formatStrValueImpl() {}
func (FormatStrPart) formatStrValueImpl()  {}

type FormatStr struct {
	Values []FormatStrValue
}

type File struct {
	name  string // relative or absolute path of the file.
	base  int    // offset of all positions (Pos) in this file in the FileSet that this File belongs to.
	size  int    // size of the file, in bytes.
	lines []int  // offsets of the first character in each line.
}

func (f *File) Name() string { return f.name }

func (f *File) AddLine(offset int) {
	f.lines = append(f.lines, offset)
}

type FileSet struct {
	base  int // base for the next file
	files []*File
}

func NewFileSet() *FileSet {
	return &FileSet{}
}

func (fs *FileSet) AddFile(name string, size int) *File {
	f := &File{name: name, base: fs.base, size: size, lines: []int{0}}
	fs.files = append(fs.files, f)
	fs.base += size
	return f
}
