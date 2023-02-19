package gokonfi

import (
	"fmt"
	"log"
	"strconv"

	"github.com/dnswlt/gokonfi/token"
)

type Parser struct {
	tokens  []token.Token
	current int
}

// Returns a new Parser that will process tokens, which will typically
// have been generated using a [Scanner].
func NewParser(tokens []token.Token) Parser {
	return Parser{tokens: tokens, current: 0}
}

// ParseError is the error type returned by [Parser] methods.
type ParseError struct {
	tok token.Token
	msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError: %s at position %d", e.msg, e.tok.Pos)
}

func (e *ParseError) Pos() token.Pos {
	return e.tok.Pos
}

type Module struct {
	expr Expr
}

type Node interface {
	token.Poser
	End() token.Pos
}

type Expr interface {
	Node
	exprNode()
}

type BinaryExpr struct {
	X     Expr
	OpPos token.Pos
	Op    token.TokenType
	Y     Expr
}

type UnaryExpr struct {
	X     Expr
	OpPos token.Pos
	Op    token.TokenType
}

// if then else
type ConditionalExpr struct {
	Cond Expr
	X    Expr
	Y    Expr
}

// func (x, y) { x + y - 1 }
type FuncExpr struct {
	Params  []AnnotatedIdent
	FuncPos token.Pos
	FuncEnd token.Pos
	Body    Expr
}

type CallExpr struct {
	Func    Expr
	Args    []Expr
	ArgsEnd token.Pos
}

type VarExpr struct {
	Name    string
	NamePos token.Pos
	NameEnd token.Pos
}

type FieldAcc struct {
	X       Expr
	Name    string
	NameEnd token.Pos
}

type AnnotatedIdent struct {
	Name    string
	NamePos token.Pos
	T       TypeAnnotation
}

// f: expr
type RecField struct {
	AnnotatedIdent
	X Expr
}

// let x: expr
type LetVar struct {
	Name    string
	NamePos token.Pos
	X       Expr
}

// { a: 1 b: "two" }
type RecExpr struct {
	LetVars map[string]LetVar
	Fields  map[string]RecField
	RecPos  token.Pos
	RecEnd  token.Pos
}

// [1, 2, 3]
type ListExpr struct {
	Elements []Expr
	ListPos  token.Pos
	ListEnd  token.Pos
}

// X :: int
type TypedExpr struct {
	X Expr
	T TypeAnnotation
}

type LiteralPos struct {
	ValPos token.Pos
	ValEnd token.Pos
}

type IntLiteral struct {
	Val int64
	LiteralPos
}

type DoubleLiteral struct {
	Val float64
	LiteralPos
}

type BoolLiteral struct {
	Val bool
	LiteralPos
}

type StrLiteral struct {
	Val string
	LiteralPos
}

type NilLiteral struct {
	LiteralPos
}

// Implementations of Expr.

func (e *BinaryExpr) Pos() token.Pos { return e.X.Pos() }
func (e *BinaryExpr) End() token.Pos { return e.Y.End() }
func (e *BinaryExpr) exprNode()      {}

func (e *UnaryExpr) Pos() token.Pos { return e.OpPos }
func (e *UnaryExpr) End() token.Pos { return e.X.End() }
func (e *UnaryExpr) exprNode()      {}

func (e *FieldAcc) Pos() token.Pos { return e.X.Pos() }
func (e *FieldAcc) End() token.Pos { return e.NameEnd }
func (e *FieldAcc) exprNode()      {}

func (e *IntLiteral) Pos() token.Pos { return e.ValPos }
func (e *IntLiteral) End() token.Pos { return e.ValEnd }
func (e *IntLiteral) exprNode()      {}

func (e *DoubleLiteral) Pos() token.Pos { return e.ValPos }
func (e *DoubleLiteral) End() token.Pos { return e.ValEnd }
func (e *DoubleLiteral) exprNode()      {}

func (e *BoolLiteral) Pos() token.Pos { return e.ValPos }
func (e *BoolLiteral) End() token.Pos { return e.ValEnd }
func (e *BoolLiteral) exprNode()      {}

func (e *StrLiteral) Pos() token.Pos { return e.ValPos }
func (e *StrLiteral) End() token.Pos { return e.ValEnd }
func (e *StrLiteral) exprNode()      {}

func (e *NilLiteral) Pos() token.Pos { return e.ValPos }
func (e *NilLiteral) End() token.Pos { return e.ValEnd }
func (e *NilLiteral) exprNode()      {}

func (e *VarExpr) Pos() token.Pos { return e.NamePos }
func (e *VarExpr) End() token.Pos { return e.NameEnd }
func (e *VarExpr) exprNode()      {}

func (e *RecExpr) Pos() token.Pos { return e.RecPos }
func (e *RecExpr) End() token.Pos { return e.RecPos }
func (e *RecExpr) exprNode()      {}

func (e *ListExpr) Pos() token.Pos { return e.ListPos }
func (e *ListExpr) End() token.Pos { return e.ListEnd }
func (e *ListExpr) exprNode()      {}

func (e *TypedExpr) Pos() token.Pos { return e.X.Pos() }
func (e *TypedExpr) End() token.Pos { return e.T.End() }
func (e *TypedExpr) exprNode()      {}

func (e *ConditionalExpr) Pos() token.Pos { return e.Cond.Pos() }
func (e *ConditionalExpr) End() token.Pos { return e.Y.End() }
func (e *ConditionalExpr) exprNode()      {}

func (e *CallExpr) Pos() token.Pos { return e.Func.Pos() }
func (e *CallExpr) End() token.Pos { return e.ArgsEnd }
func (e *CallExpr) exprNode()      {}

func (e *FuncExpr) Pos() token.Pos { return e.FuncPos }
func (e *FuncExpr) End() token.Pos { return e.FuncEnd }
func (e *FuncExpr) exprNode()      {}

// Type annotations.

type TypeAnnotation interface {
	Node
	TypeId() string
	typeAnnotationImpl()
}

type NamedType struct {
	Name    string
	NamePos token.Pos
	NameEnd token.Pos
}

// Implementations of TypeAnnotation.

func (t *NamedType) TypeId() string      { return t.Name }
func (t *NamedType) typeAnnotationImpl() {}
func (t *NamedType) Pos() token.Pos      { return t.NamePos }
func (t *NamedType) End() token.Pos      { return t.NameEnd }

// Parser methods.

func (p *Parser) advance() token.Token {
	if !p.AtEnd() {
		p.current++
	}
	return p.previous()
}

// previous returns the Token most recently returned by advance.
func (p *Parser) previous() token.Token {
	if p.current == 0 {
		return token.Token{}
	}
	return p.tokens[p.current-1]
}

func (p *Parser) peek() token.Token {
	return p.tokens[p.current]
}

func (p *Parser) match(tokenTypes ...token.TokenType) bool {
	t := p.peek()
	for _, typ := range tokenTypes {
		if t.Typ == typ {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) expect(tokenType token.TokenType, context string) error {
	if !p.match(tokenType) {
		t := p.peek()
		return &ParseError{
			tok: t, msg: fmt.Sprintf("expected token of type %s, got '%s' (%s) in %s",
				tokenType, t.Val, t.Typ, context)}
	}
	return nil
}

func (p *Parser) fail(msg string, fmtArgs ...any) error {
	return &ParseError{tok: p.peek(), msg: fmt.Sprintf(msg, fmtArgs...)}
}

func (p *Parser) failat(t token.Token, msg string, fmtArgs ...any) error {
	return &ParseError{tok: t, msg: fmt.Sprintf(msg, fmtArgs...)}
}

// AtEnd returns true if the parser has processed all tokens.
func (p *Parser) AtEnd() bool {
	return p.current >= len(p.tokens)-1
}

func ParseModule(input string, file *token.File) (*Module, error) {
	ts, err := NewScanner(string(input), file).ScanAll()
	if err != nil {
		return nil, err
	}
	p := NewParser(ts)
	// For now, a module is simply an expression.
	e, err := p.Expression()
	if err != nil {
		return nil, err
	}
	return &Module{expr: e}, nil
}

// Parses an expression.
func (p *Parser) Expression() (Expr, error) {
	return p.conditional()
}

func (p *Parser) conditional() (Expr, error) {
	if p.match(token.If) {
		cond, err := p.Expression()
		if err != nil {
			return nil, err
		}
		if err = p.expect(token.Then, "conditional"); err != nil {
			return nil, err
		}
		x, err := p.Expression()
		if err != nil {
			return nil, err
		}
		if err = p.expect(token.Else, "conditional"); err != nil {
			return nil, err
		}
		y, err := p.Expression()
		if err != nil {
			return nil, err
		}
		return &ConditionalExpr{cond, x, y}, nil
	}
	return p.logicalOr()
}

// logical_or     -> logical_and ( "||" logical_and )* ;
func (p *Parser) logicalOr() (Expr, error) {
	x, err := p.logicalAnd()
	if err != nil {
		return nil, err
	}
	for p.match(token.LogicalOr) {
		t := p.previous()
		y, err := p.logicalAnd()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{X: x, OpPos: t.Pos, Op: t.Typ, Y: y}
	}
	return x, nil
}

// logical_and    -> comparison ( "&&" comparison )* ;
func (p *Parser) logicalAnd() (Expr, error) {
	x, err := p.comparison()
	if err != nil {
		return nil, err
	}
	for p.match(token.LogicalAnd) {
		t := p.previous()
		y, err := p.comparison()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{X: x, OpPos: t.Pos, Op: t.Typ, Y: y}
	}
	return x, nil
}

// comparison     -> term ( ( "!=" | "==" | ">" | ">=" | "<" | "<=" ) term )* ;
func (p *Parser) comparison() (Expr, error) {
	x, err := p.term()
	if err != nil {
		return nil, err
	}
	for p.match(token.NotEqual, token.Equal, token.GreaterThan, token.GreaterEq, token.LessThan, token.LessEq) {
		t := p.previous()
		y, err := p.term()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{X: x, OpPos: t.Pos, Op: t.Typ, Y: y}
	}
	return x, nil
}

// term           -> factor ( ( "-" | "+" | "|" | "^" | "@" ) factor )* ;
func (p *Parser) term() (Expr, error) {
	x, err := p.factor()
	if err != nil {
		return nil, err
	}
	for p.match(token.Minus, token.Plus, token.BitwiseOr, token.BitwiseXor, token.Merge) {
		t := p.previous()
		y, err := p.factor()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{X: x, OpPos: t.Pos, Op: t.Typ, Y: y}
	}
	return x, nil
}

// factor         -> unary ( ( "/" | "*" | "%" | "<<" | ">>" | "&" ) unary )* ;
func (p *Parser) factor() (Expr, error) {
	x, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.match(token.Div, token.Times, token.Modulo, token.ShiftLeft, token.ShiftRight, token.BitwiseAnd) {
		t := p.previous()
		y, err := p.unary()
		if err != nil {
			return nil, err
		}
		x = &BinaryExpr{X: x, OpPos: t.Pos, Op: t.Typ, Y: y}
	}
	return x, nil
}

// unary          -> ( "!" | "-" ) unary
//
//	| annotated_primary ;
func (p *Parser) unary() (Expr, error) {
	if p.match(token.Minus, token.Complement, token.Not) {
		t := p.previous()
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{X: x, OpPos: t.Pos, Op: t.Typ}, nil
	}
	return p.annotatedPrimary()
}

func (p *Parser) annotatedPrimary() (Expr, error) {
	e, err := p.primary()
	if err != nil {
		return nil, err
	}
	if p.match(token.OfType) {
		typ, err := p.typeAnnotation()
		if err != nil {
			return nil, err
		}
		e = &TypedExpr{X: e, T: typ}
	}
	return e, nil
}

// annotated_primary   -> primary ["::" type_annotation]
func (p *Parser) primary() (Expr, error) {
	op, err := p.operand()
	if err != nil {
		return nil, err
	}
	e := op
	// Parse optional postfix ("." field | "(" argList ")" | "[" expr "]" | "::" type_annotation )
Loop:
	for !p.AtEnd() {
		switch {
		case p.match(token.Dot):
			if !p.match(token.Ident) {
				return nil, &ParseError{tok: p.peek(), msg: fmt.Sprintf("expected identifier, got %s", p.peek().Typ)}
			}
			ident := p.previous()
			e = &FieldAcc{X: e, Name: ident.Val, NameEnd: ident.End}
		case p.match(token.LeftParen):
			args, err := p.exprList(token.Comma, token.RightParen)
			if err != nil {
				return nil, err
			}
			e = &CallExpr{Func: e, Args: args, ArgsEnd: p.previous().End}
		default:
			break Loop
		}
	}
	return e, nil
}

func (p *Parser) exprList(sep token.TokenType, close token.TokenType) ([]Expr, error) {
	args := []Expr{}
	if p.match(close) {
		return args, nil
	}
	for !p.AtEnd() {
		e, err := p.Expression()
		if err != nil {
			return nil, err
		}
		args = append(args, e)
		if p.match(close) {
			return args, nil
		}
		if err = p.expect(sep, "expression list"); err != nil {
			return nil, err
		}
	}
	return nil, &ParseError{tok: p.previous(), msg: "reached end of input while parsing expression list"}
}

func (p *Parser) identList(sep token.TokenType, close token.TokenType) ([]AnnotatedIdent, error) {
	idents := []AnnotatedIdent{}
	seen := make(map[string]bool)
	if p.match(close) {
		return idents, nil
	}
	for !p.AtEnd() {
		ident, err := p.annotatedIdent()
		if err != nil {
			return nil, err
		}
		if seen[ident.Name] {
			return nil, p.failat(p.previous(), "duplicate identifier in identifier list: %s", ident.Name)
		}
		seen[ident.Name] = true
		idents = append(idents, ident)
		if p.match(close) {
			return idents, nil
		}
		if err := p.expect(sep, "identifier list"); err != nil {
			return nil, err
		}
	}
	return nil, &ParseError{tok: p.previous(), msg: "reached end of input while parsing identifier list"}
}

// Operands are syntactically closed expressions, i.e. either single tokens or well-delimited
// expressions like "(" <expr> ")".
func (p *Parser) operand() (Expr, error) {
	switch {
	case p.match(token.LeftParen):
		e, err := p.Expression()
		if err != nil {
			return nil, err
		}
		if err = p.expect(token.RightParen, "operand"); err != nil {
			return nil, err
		}
		return e, nil
	case p.match(token.BoolLiteral):
		t := p.previous()
		b := true
		if t.Val == "false" {
			b = false
		}
		return &BoolLiteral{Val: b, LiteralPos: LiteralPos{t.Pos, t.End}}, nil
	case p.match(token.IntLiteral):
		t := p.previous()
		x, err := strconv.ParseInt(t.Val, 10, 64)
		if err != nil {
			return nil, err
		}
		return &IntLiteral{Val: x, LiteralPos: LiteralPos{t.Pos, t.End}}, nil
	case p.match(token.DoubleLiteral):
		t := p.previous()
		x, err := strconv.ParseFloat(t.Val, 64)
		if err != nil {
			return nil, err
		}
		return &DoubleLiteral{Val: x, LiteralPos: LiteralPos{t.Pos, t.End}}, nil
	case p.match(token.StrLiteral):
		t := p.previous()
		return &StrLiteral{Val: t.Val, LiteralPos: LiteralPos{t.Pos, t.End}}, nil
	case p.match(token.FormatStrLiteral):
		t := p.previous()
		if t.Fmt == nil || len(t.Fmt.Values) == 0 {
			log.Fatalf("empty .Fmt in FormatStrLiteral at %d", t.Pos)
		}
		// Format strings are explicitly desugared in the AST:
		// "prefix${expr}suffix" ==> "prefix" + str(expr) + "suffix"
		plusArgs := make([]Expr, len(t.Fmt.Values))
		for i, fmtValue := range t.Fmt.Values {
			switch v := fmtValue.(type) {
			case token.FormatStrPart:
				plusArgs[i] = &StrLiteral{Val: v.Val, LiteralPos: LiteralPos{v.Pos, v.End}}
			case token.FormattedValue:
				if len(v.Tokens) == 0 {
					// Interpret ${} as an empty string.
					plusArgs[i] = &StrLiteral{Val: "", LiteralPos: LiteralPos{v.Pos, v.End}}
					continue
				}
				cp := NewParser(v.Tokens)
				fe, err := cp.Expression()
				if err != nil {
					return nil, err
				}
				if !cp.AtEnd() {
					return nil, &ParseError{tok: cp.peek(), msg: "remaining tokens in interpolated expression"}
				}
				plusArgs[i] =
					&CallExpr{
						Func: &VarExpr{Name: "str", NamePos: v.Pos, NameEnd: v.Pos},
						Args: []Expr{fe}, ArgsEnd: v.End}
			}
		}
		fmtExpr := plusArgs[0]
		for _, plusArg := range plusArgs[1:] {
			fmtExpr = &BinaryExpr{X: fmtExpr, Y: plusArg, Op: token.Plus, OpPos: plusArg.Pos()}
		}
		return fmtExpr, nil
	case p.match(token.Nil):
		t := p.previous()
		return &NilLiteral{LiteralPos: LiteralPos{t.Pos, t.End}}, nil
	case p.match(token.Ident):
		t := p.previous()
		return &VarExpr{Name: t.Val, NamePos: t.Pos, NameEnd: t.End}, nil
	case p.peek().Typ == token.LeftBrace:
		// Record
		r, err := p.record()
		if err != nil {
			return nil, err
		}
		return r, nil
	case p.match(token.LeftSquare):
		start := p.previous()
		// List
		xs, err := p.exprList(token.Comma, token.RightSquare)
		if err != nil {
			return nil, err
		}
		return &ListExpr{Elements: xs, ListPos: start.Pos, ListEnd: p.previous().End}, nil
	case p.match(token.Func):
		funcPos := p.previous().Pos
		if err := p.expect(token.LeftParen, "func"); err != nil {
			return nil, err
		}
		params, err := p.identList(token.Comma, token.RightParen)
		if err != nil {
			return nil, err
		}
		if err = p.expect(token.LeftBrace, "func"); err != nil {
			return nil, err
		}
		body, err := p.Expression()
		if err != nil {
			return nil, err
		}
		if err = p.expect(token.RightBrace, "func"); err != nil {
			return nil, err
		}
		return &FuncExpr{Params: params, FuncPos: funcPos, FuncEnd: p.previous().End, Body: body}, nil
	case p.match(token.Template):
		// Templates are syntactic sugar for functions.
		funcPos := p.previous().Pos
		if err := p.expect(token.LeftParen, "template"); err != nil {
			return nil, err
		}
		params, err := p.identList(token.Comma, token.RightParen)
		if err != nil {
			return nil, err
		}
		body, err := p.record()
		if err != nil {
			return nil, err
		}
		return &FuncExpr{Params: params, FuncPos: funcPos, FuncEnd: p.previous().End, Body: body}, nil
	}
	return nil, p.fail("unexpected token type %s for primary expression", p.peek().Typ)
}

func (p *Parser) record() (Expr, error) {
	if !p.match(token.LeftBrace) {
		return nil, p.fail("expected '{' token to parse record, got %s", p.peek().Val)
	}
	recPos := p.previous().Pos
	letVars := make(map[string]LetVar)
	fields := make(map[string]RecField)
	seen := make(map[string]bool)
	for !p.AtEnd() {
		if p.match(token.RightBrace) {
			return &RecExpr{LetVars: letVars, Fields: fields, RecPos: recPos, RecEnd: p.previous().End}, nil
		}
		fTok := p.peek()
		if fTok.Typ == token.Let {
			l, err := p.letVar()
			if err != nil {
				return nil, err
			}
			if seen[l.Name] {
				return nil, &ParseError{tok: fTok, msg: fmt.Sprintf("duplicate let binding field '%s'", l.Name)}
			}
			seen[l.Name] = true
			letVars[l.Name] = *l
		} else {
			f, err := p.recordField()
			if err != nil {
				return nil, err
			}
			if seen[f.Name] {
				return nil, &ParseError{tok: fTok, msg: fmt.Sprintf("duplicate record field '%s'", f.Name)}
			}
			seen[f.Name] = true
			fields[f.Name] = *f
		}
	}
	return nil, p.fail("reached end of input while parsing record")
}

// Can be one of
// "let" <ident> ":" <expr>
// "let" <ident> "(" <id_list> ")" ":" <expr>
// "let" "template" <ident> "(" <id_list> ")" <record>
//
// Examples:
// let x: 7
//
// let f(x): 17 + x
// <==>
// let f: func (x) { 17 + x }
//
// let template f(x, y) { x: 7 }
// <==>
// let f: template (x, y) { x: 7 }
// <==>
// let f: func (x, y) { { x: 7 } }
func (p *Parser) letVar() (*LetVar, error) {
	if err := p.expect(token.Let, "let"); err != nil {
		return nil, err
	}
	switch {
	case p.match(token.Ident):
		v := p.previous()
		if p.match(token.LeftParen) {
			// let function definition
			params, err := p.identList(token.Comma, token.RightParen)
			if err != nil {
				return nil, err
			}
			if err = p.expect(token.Colon, "func"); err != nil {
				return nil, err
			}
			body, err := p.Expression()
			if err != nil {
				return nil, err
			}
			f := &FuncExpr{Params: params, FuncPos: v.Pos, FuncEnd: body.End(), Body: body}
			return &LetVar{Name: v.Val, NamePos: v.Pos, X: f}, nil
		}
		// Regular variable binding
		if err := p.expect(token.Colon, "let"); err != nil {
			return nil, err
		}
		expr, err := p.Expression()
		if err != nil {
			return nil, err
		}
		return &LetVar{Name: v.Val, NamePos: v.Pos, X: expr}, nil
	case p.match(token.Template):
		if err := p.expect(token.Ident, "template"); err != nil {
			return nil, err
		}
		v := p.previous()
		if err := p.expect(token.LeftParen, "template"); err != nil {
			return nil, err
		}
		params, err := p.identList(token.Comma, token.RightParen)
		if err != nil {
			return nil, err
		}
		body, err := p.record()
		if err != nil {
			return nil, err
		}
		f := &FuncExpr{Params: params, FuncPos: v.Pos, FuncEnd: body.End(), Body: body}
		return &LetVar{Name: v.Val, NamePos: v.Pos, X: f}, nil
	}
	return nil, &ParseError{tok: p.peek(), msg: fmt.Sprintf("unexpected token '%s' in let binding", p.peek().Val)}
}

func (p *Parser) recordField() (*RecField, error) {
	v, err := p.annotatedIdent()
	if err != nil {
		return nil, err
	}
	if !p.match(token.Colon) {
		t := p.peek()
		return nil, &ParseError{tok: t, msg: fmt.Sprintf("expected ':' for record field, got %s", t.Typ)}
	}
	expr, err := p.Expression()
	if err != nil {
		return nil, err
	}
	return &RecField{AnnotatedIdent: v, X: expr}, nil
}

func (p *Parser) typeAnnotation() (TypeAnnotation, error) {
	// For now, only type names, no complex expressions.
	if p.match(token.Ident) {
		t := p.previous()
		return &NamedType{Name: t.Val, NamePos: t.Pos, NameEnd: t.End}, nil
	}
	return nil, p.fail("typeAnnotation: unexpected token")
}

func (p *Parser) annotatedIdent() (AnnotatedIdent, error) {
	if err := p.expect(token.Ident, "annotatedIdent"); err != nil {
		return AnnotatedIdent{}, err
	}
	ident := p.previous()
	var typ TypeAnnotation
	if p.match(token.OfType) {
		t, err := p.typeAnnotation()
		if err != nil {
			return AnnotatedIdent{}, err
		}
		typ = t
	}
	return AnnotatedIdent{Name: ident.Val, T: typ, NamePos: ident.Pos}, nil
}
