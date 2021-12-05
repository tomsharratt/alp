package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/tomsharratt/alp/lexer"
	"github.com/tomsharratt/alp/token"
)

const PROMT = ">> "

func Run(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprintf(out, "%s", PROMT)
		if !scanner.Scan() {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)

		for t := l.NextToken(); t.Type != token.EOF; t = l.NextToken() {
			fmt.Fprintf(out, "%+v\n", t)
		}
	}
}
