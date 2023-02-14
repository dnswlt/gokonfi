package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dnswlt/gokonfi"
)

var (
	printResult  bool
	outputFormat string
)

func init() {
	flag.StringVar(&outputFormat, "format", "json", "output format")
	flag.BoolVar(&printResult, "p", true, "print result to stdout")
}

func evalFileAsExpression(filename string) (gokonfi.Val, error) {
	input, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	ts, err := gokonfi.NewScanner(string(input), nil).ScanAll()
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
