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

type ParseError struct {
	tok token.Token
	msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("ParseError: %s at position %d", e.msg, e.tok.Pos)
}

type Node interface {
	Pos() token.Pos
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

type IntLiteral struct {
	Val    int64
	ValPos token.Pos
	ValEnd token.Pos
}

type BoolLiteral struct {
	Val    bool
	ValPos token.Pos
	ValEnd token.Pos
}

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

func (e *IntLiteral) Pos() token.Pos {
	return e.ValPos
}

func (e *IntLiteral) End() token.Pos {
	return e.ValEnd
}

func (e *IntLiteral) exprNode() {}

func (e *BoolLiteral) Pos() token.Pos {
	return e.ValPos
}

func (e *BoolLiteral) End() token.Pos {
	return e.ValEnd
}

func (e *BoolLiteral) exprNode() {}

func NewParser(tokens []token.Token) Parser {
	return Parser{tokens: tokens, current: 0}
}

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

func (p *Parser) AtEnd() bool {
	return p.current >= len(p.tokens)-1
}

/*

Precedence    Operator
    5             *  /  %  <<  >>  &  &^
    4             +  -  |  ^
    3             ==  !=  <  <=  >  >=
    2             &&
    1             ||

expression     -> logical_or ;
logical_or     -> logical_and ( "||" logical_and )* ;
logical_and    -> comparison ( "&&" comparison )* ;
comparison     -> term ( ( "!=" | "==" | ">" | ">=" | "<" | "<=" ) term )* ;
term           -> factor ( ( "-" | "+" | "|" | "^" ) factor )* ;
factor         -> unary ( ( "/" | "*" | "%" | "<<" | ">>" | "&" ) unary )* ;
unary          -> ( "!" | "-" ) unary
               | primary ;
primary        -> NUMBER | STRING | "true" | "false" | "nil"
               | "(" expression ")" ;
*/

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

// term           -> factor ( ( "-" | "+" | "|" | "^" ) factor )* ;
func (p *Parser) term() (Expr, error) {
	x, err := p.factor()
	if err != nil {
		return nil, err
	}
	for p.match(token.Minus, token.Plus, token.BitwiseOr, token.BitwiseXor) {
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

// primary        -> NUMBER | STRING | "true" | "false" | "nil"
//
//	| "(" expression ")" ;
func (p *Parser) primary() (Expr, error) {
	if p.match(token.LeftParen) {
		e, err := p.Expression()
		if err != nil {
			return nil, err
		}
		if !p.match(token.RightParen) {
			return nil, &ParseError{tok: p.previous(), msg: fmt.Sprintf("Expected ')', got %s", p.previous().Val)}
		}
		return e, nil
	}
	if p.match(token.BoolLiteral) {
		t := p.previous()
		b := true
		if t.Val == "false" {
			b = false
		}
		return &BoolLiteral{Val: b, ValPos: t.Pos, ValEnd: t.End}, nil
	}
	if p.match(token.IntLiteral) {
		t := p.previous()
		x, err := strconv.ParseInt(t.Val, 10, 64)
		if err != nil {
			return nil, err
		}
		return &IntLiteral{Val: x, ValPos: t.Pos, ValEnd: t.End}, nil
	}
	return nil, &ParseError{tok: p.peek(), msg: fmt.Sprintf("Unexpected token type %s for primary expression", p.peek().Typ)}
}
