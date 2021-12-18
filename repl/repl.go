package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/tomsharratt/alp/evaluator"
	"github.com/tomsharratt/alp/lexer"
	"github.com/tomsharratt/alp/object"
	"github.com/tomsharratt/alp/parser"
)

const PROMT = ">> "

func Run(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	env := object.NewEnvironment()

	for {
		fmt.Fprintf(out, "%s", PROMT)
		if !scanner.Scan() {
			return
		}

		ctx := context.Background()

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		evaluated, err := evaluator.Eval(ctx, program, env)
		if err == nil && evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
