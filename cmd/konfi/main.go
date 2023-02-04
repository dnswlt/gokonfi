package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dnswlt/gokonfi"
	"github.com/dnswlt/gokonfi/token"
)

var (
	printResult  bool
	outputFormat string
)

func init() {
	flag.StringVar(&outputFormat, "format", "json", "output format")
	flag.BoolVar(&printResult, "p", true, "print result to stdout")
}

func scanTokens(input string) ([]token.Token, error) {
	s := gokonfi.NewScanner(input)
	r := []token.Token{}
	for {
		t, err := s.NextToken()
		if err != nil {
			return nil, err
		}
		r = append(r, t)
		if t.Typ == token.EndOfInput {
			break
		}
	}
	return r, nil
}

func evalFileAsExpression(filename string) (gokonfi.Val, error) {
	input, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	ts, err := scanTokens(string(input))
	if err != nil {
		return nil, err
	}
	p := gokonfi.NewParser(ts)
	expr, err := p.Expression()
	if err != nil {
		return nil, err
	}
	if !p.AtEnd() {
		return nil, fmt.Errorf("did not parse entire input")
	}
	return gokonfi.Eval(expr, gokonfi.GlobalCtx())
}

func run() error {
	flag.Parse()
	if len(flag.Args()) != 1 {
		return fmt.Errorf("expected one input file, got %d", len(flag.Args()))
	}
	filename := flag.Arg(0)
	val, err := evalFileAsExpression(filename)
	if err != nil {
		return fmt.Errorf("failed to process %s: %s", filename, err)
	}
	switch outputFormat {
	case "json":
		js, err := gokonfi.EncodeAsJsonIndent(val)
		if err != nil {
			return err
		}
		fmt.Println(js)
	default:
		return fmt.Errorf("unknown output format: %s", outputFormat)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
