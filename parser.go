package gokonfi

import "github.com/dnswlt/gokonfi/token"

type Parser struct {
	scanner   Scanner
	lookahead []token.Token
}

type Node interface {
	Pos() int
	End() int
}

type Expr interface {
	Node
	exprNode()
}

type BinaryExpr struct {
	X     Expr
	OpPos int
	Op    token.TokenType
	Y     Expr
}

func (e *BinaryExpr) Pos() int {
	return e.X.Pos()
}

func (e *BinaryExpr) End() int {
	return e.Y.End()
}

func (e *BinaryExpr) exprNode() {}

func NewParser(s Scanner) Parser {
	return Parser{scanner: s}
}

func (p *Parser) advance() (token.Token, error) {
	if len(p.lookahead) > 0 {
		cur := p.lookahead[0]
		p.lookahead = p.lookahead[1:]
		return cur, nil
	}
	return p.scanner.NextToken()
}

func (p *Parser) peek() (token.Token, error) {
	if len(p.lookahead) > 0 {
		return p.lookahead[0], nil
	}
	t, err := p.scanner.NextToken()
	if err == nil {
		p.lookahead = append(p.lookahead, t)
	}
	return t, err
}

func (p *Parser) AtEnd() bool {
	t, _ := p.peek()
	return t.Typ == token.EndOfInput
}

func (p *Parser) Expression() (Expr, error) {
	return &BinaryExpr{X: nil, OpPos: 0, Op: token.PlusOp, Y: nil}, nil
}
