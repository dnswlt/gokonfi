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
	flag.StringVar(&outputFormat, "format", "yaml", "output format (supported: yaml, json)")
	flag.BoolVar(&printResult, "p", true, "print result to stdout")
}

func run() error {
	flag.Parse()
	if len(flag.Args()) != 1 {
		return fmt.Errorf("expected one input file, got %d", len(flag.Args()))
	}
	filename := flag.Arg(0)
	ctx := gokonfi.GlobalCtx()
	mod, err := gokonfi.LoadModule(filename, ctx)
	if err != nil {
		return gokonfi.FormattedError(err, ctx)
	}
	switch outputFormat {
	case "json":
		js, err := gokonfi.EncodeAsJsonIndent(mod.Body())
		if err != nil {
			return err
		}
		fmt.Println(js)
	case "yaml":
		yml, err := gokonfi.EncodeAsYaml(mod.Body())
		if err != nil {
			return err
		}
		fmt.Print(yml) // yml always ends in a newline.
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
