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

func run() error {
	flag.Parse()
	if len(flag.Args()) != 1 {
		return fmt.Errorf("expected one input file, got %d", len(flag.Args()))
	}
	filename := flag.Arg(0)
	ctx := gokonfi.GlobalCtx()
	val, err := gokonfi.LoadModule(filename, ctx)
	if err != nil {
		return ctx.FormattedError(err)
	}
	switch outputFormat {
	case "json":
		js, err := gokonfi.EncodeAsJsonIndent(val.Body())
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
