//nolint:gosec
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/sanity-io/go-groq/ast"
	"github.com/sanity-io/go-groq/parser"
	"github.com/sanity-io/go-groq/print"
)

var opts struct {
	Output        string `short:"o" long:"output" description:"Write to file instead of standard output" value-name:"FILE"`
	WriteToSource bool   `short:"w" description:"Write back to input file"`
	Compact       bool   `short:"c" long:"compact" description:"Compact syntax"`
}

func main() {
	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	parser.Usage = "[- | FILE]"

	args, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return processFile("-")
	}
	for _, arg := range args {
		if err := processFile(arg); err != nil {
			return fmt.Errorf("formatting %s: %w", arg, err)
		}
	}
	return nil
}

func processFile(fileName string) error {
	query, err := readFile(fileName)
	if err != nil {
		return err
	}

	formatted, err := format(query)
	if err != nil {
		return err
	}

	if opts.Output != "" {
		return ioutil.WriteFile(opts.Output, []byte(formatted), 0644)
	}

	if opts.WriteToSource {
		if opts.Output != "" {
			return errors.New("cannot use -w together with -o flag")
		}
		if fileName == "-" {
			return errors.New("cannot use -w with stdin")
		}
		return ioutil.WriteFile(fileName, []byte(formatted), 0644)
	}

	_, err = os.Stdout.Write([]byte(formatted))
	return err
}

func readFile(fileName string) (string, error) {
	if fileName == "-" {
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func format(query string) (string, error) {
	q, err := parser.Parse(query, parser.WithParamNodes())
	if err != nil {
		return "", fmt.Errorf("parsing query: %w", err)
	}

	var printer func(query ast.Expression, w io.Writer) error
	if opts.Compact {
		printer = print.Print
	} else {
		printer = print.PrettyPrint
	}

	var buf bytes.Buffer
	if err := printer(q, &buf); err != nil {
		return "", fmt.Errorf("formatting query: %w", err)
	}
	if _, err := buf.Write([]byte("\n")); err != nil {
		return "", err
	}
	return buf.String(), nil
}
