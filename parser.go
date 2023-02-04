package gokonfi

import (
	"fmt"
	"strconv"

	"github.com/dnswlt/gokonfi/token"
)

type Parser struct {
	tokens  []token.Token
	current int
}

func NewParser(tokens []token.Token) Parser {
	return Parser{tokens: tokens, current: 0}
}

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

type ConditionalExpr struct {
	Cond Expr
	X    Expr
	Y    Expr
}

type FuncExpr struct {
	// func (x, y) { x + y - 1 }
	Params  []*VarExpr
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

type RecField struct {
	Name    string
	NamePos token.Pos
	Val     Expr
}

type LetVar struct {
	Name    string
	NamePos token.Pos
	Val     Expr
}

type RecExpr struct {
	LetVars map[string]LetVar
	Fields  map[string]RecField
	RecPos  token.Pos
	RecEnd  token.Pos
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

func (e *BinaryExpr) Pos() token.Pos {
	return e.X.Pos()
}
func (e *BinaryExpr) End() token.Pos {
	return e.Y.End()
}
func (e *BinaryExpr) exprNode() {}

func (e *UnaryExpr) Pos() token.Pos {
	return e.OpPos
}
func (e *UnaryExpr) End() token.Pos {
	return e.X.End()
}
func (e *UnaryExpr) exprNode() {}

func (e *FieldAcc) Pos() token.Pos {
	return e.X.Pos()
}
func (e *FieldAcc) End() token.Pos {
	return e.NameEnd
}
func (e *FieldAcc) exprNode() {}

func (e *IntLiteral) Pos() token.Pos {
	return e.ValPos
}
func (e *IntLiteral) End() token.Pos {
	return e.ValEnd
}
func (e *IntLiteral) exprNode() {}

func (e *DoubleLiteral) Pos() token.Pos {
	return e.ValPos
}
func (e *DoubleLiteral) End() token.Pos {
	return e.ValEnd
}
func (e *DoubleLiteral) exprNode() {}

func (e *BoolLiteral) Pos() token.Pos {
	return e.ValPos
}
func (e *BoolLiteral) End() token.Pos {
	return e.ValEnd
}
func (e *BoolLiteral) exprNode() {}

func (e *StrLiteral) Pos() token.Pos {
	return e.ValPos
}
func (e *StrLiteral) End() token.Pos {
	return e.ValEnd
}
func (e *StrLiteral) exprNode() {}

func (e *NilLiteral) Pos() token.Pos {
	return e.ValPos
}
func (e *NilLiteral) End() token.Pos {
	return e.ValEnd
}
func (e *NilLiteral) exprNode() {}

func (e *VarExpr) Pos() token.Pos {
	return e.NamePos
}
func (e *VarExpr) End() token.Pos {
	return e.NameEnd
}
func (e *VarExpr) exprNode() {}

func (e *RecExpr) Pos() token.Pos {
	return e.RecPos
}
func (e *RecExpr) End() token.Pos {
	return e.RecPos
}
func (e *RecExpr) exprNode() {}

func (e *ConditionalExpr) Pos() token.Pos {
	return e.Cond.Pos()
}
func (e *ConditionalExpr) End() token.Pos {
	return e.Y.End()
}
func (e *ConditionalExpr) exprNode() {}

func (e *CallExpr) Pos() token.Pos {
	return e.Func.Pos()
}
func (e *CallExpr) End() token.Pos {
	return e.ArgsEnd
}
func (e *CallExpr) exprNode() {}

func (e *FuncExpr) Pos() token.Pos {
	return e.FuncPos
}
func (e *FuncExpr) End() token.Pos {
	return e.FuncEnd
}
func (e *FuncExpr) exprNode() {}

// Parser methods.

func (p *Parser) advance() token.Token {
	if !p.AtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) previous() token.Token {
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

func (p *Parser) AtEnd() bool {
	return p.current >= len(p.tokens)-1
}

// Parses an expression.
func (p *Parser) Expression() (Expr, error) {
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
//	| primary ;
func (p *Parser) unary() (Expr, error) {
	if p.match(token.Minus, token.Complement, token.Not) {
		t := p.previous()
		x, err := p.unary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{X: x, OpPos: t.Pos, Op: t.Typ}, nil
	}
	return p.primary()
}

func (p *Parser) primary() (Expr, error) {
	op, err := p.operand()
	if err != nil {
		return nil, err
	}
	e := op
	// Parse optional postfix ("." field | "(" argList ")" | "[" expr "]")
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

func (p *Parser) identList(sep token.TokenType, close token.TokenType) ([]*VarExpr, error) {
	idents := []*VarExpr{}
	seen := make(map[string]bool)
	if p.match(close) {
		return idents, nil
	}
	for !p.AtEnd() {
		if err := p.expect(token.Ident, "identifier list"); err != nil {
			return nil, err
		}
		t := p.previous()
		if seen[t.Val] {
			return nil, &ParseError{tok: t, msg: fmt.Sprintf("duplicate identifier in identifier list: %s", t.Val)}
		}
		seen[t.Val] = true
		idents = append(idents, &VarExpr{Name: t.Val, NamePos: t.Pos, NameEnd: t.End})
		if p.match(close) {
			return idents, nil
		}
		if err := p.expect(sep, "identifier list"); err != nil {
			return nil, err
		}
	}
	return nil, &ParseError{tok: p.previous(), msg: "reached end of input while parsing identifier list"}
}

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
	case p.match(token.NilLiteral):
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
	case p.match(token.If):
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
	return nil, &ParseError{tok: p.peek(), msg: fmt.Sprintf("unexpected token type %s for primary expression", p.peek().Typ)}
}

func (p *Parser) record() (Expr, error) {
	if !p.match(token.LeftBrace) {
		return nil, &ParseError{tok: p.peek(), msg: fmt.Sprintf("expected '{' token to parse record, got %s", p.peek().Val)}
	}
	recPos := p.previous().Pos
	letVars := make(map[string]LetVar)
	fields := make(map[string]RecField)
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
			if _, ok := letVars[l.Name]; ok {
				return nil, &ParseError{tok: fTok, msg: fmt.Sprintf("duplicate let binding field '%s'", l.Name)}
			}
			letVars[l.Name] = *l
		} else {
			f, err := p.recordField()
			if err != nil {
				return nil, err
			}
			if _, ok := fields[f.Name]; ok {
				return nil, &ParseError{tok: fTok, msg: fmt.Sprintf("duplicate record field '%s'", f.Name)}
			}
			fields[f.Name] = *f
		}
	}
	return nil, &ParseError{tok: p.previous(), msg: "reached end of input while parsing record"}
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
			return &LetVar{Name: v.Val, NamePos: v.Pos, Val: f}, nil
		}
		// Regular variable binding
		if err := p.expect(token.Colon, "let"); err != nil {
			return nil, err
		}
		expr, err := p.Expression()
		if err != nil {
			return nil, err
		}
		return &LetVar{Name: v.Val, NamePos: v.Pos, Val: expr}, nil
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
		return &LetVar{Name: v.Val, NamePos: v.Pos, Val: f}, nil
	}
	return nil, &ParseError{tok: p.peek(), msg: fmt.Sprintf("unexpected token '%s' in let binding", p.peek().Val)}
}

func (p *Parser) recordField() (*RecField, error) {
	if !p.match(token.Ident) {
		t := p.peek()
		return nil, &ParseError{tok: t, msg: fmt.Sprintf("expected identifier for record field, got %s", t.Typ)}
	}
	field := p.previous()
	if !p.match(token.Colon) {
		t := p.peek()
		return nil, &ParseError{tok: t, msg: fmt.Sprintf("expected ':' for record field, got %s", t.Typ)}
	}
	expr, err := p.Expression()
	if err != nil {
		return nil, err
	}
	return &RecField{Name: field.Val, NamePos: field.Pos, Val: expr}, nil
}
