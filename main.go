//nolint:gosec
package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/sanity-io/go-groq/parser"
	"github.com/sanity-io/go-groq/print"
)

var opts struct {
	Output        string `short:"o" long:"output" description:"Write to file instead of standard output" value-name:"FILE"`
	WriteToSource bool   `short:"w" description:"Write back to input file"`
}

func main() {
	p := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[- | FILE]"

	args, err := p.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			p.WriteHelp(os.Stdout)
			os.Exit(2)
		}
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
		os.Exit(1)
	}

	if err := run(args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %s\n", err)
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
		return os.WriteFile(opts.Output, []byte(formatted), 0644)
	}

	if opts.WriteToSource {
		if opts.Output != "" {
			return errors.New("cannot use -w together with -o flag")
		}
		if fileName == "-" {
			return errors.New("cannot use -w with stdin")
		}
		return os.WriteFile(fileName, []byte(formatted), 0644)
	}

	_, err = os.Stdout.Write([]byte(formatted))
	return err
}

func readFile(fileName string) (string, error) {
	if fileName == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	b, err := os.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func format(query string) (string, error) {
	a, err := parser.Parse(query, parser.WithParamNodes())
	if err != nil {
		return "", fmt.Errorf("parsing query: %w", err)
	}

	var buf bytes.Buffer
	if err := print.PrettyPrint(a, &buf); err != nil {
		return "", fmt.Errorf("formatting query: %w", err)
	}
	if _, err := buf.Write([]byte("\n")); err != nil {
		return "", err
	}
	return buf.String(), nil
}
