package ast

import (
	"testing"

	"github.com/tomsharratt/alp/token"
)

func TestString(t *testing.T) {
	program := &Program{
		Statements: []Statement{
			&LetStatement{
				Token: token.Token{Type: token.LET, Literal: "let"},
				Name: &Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "myVar"},
					Value: "myVar",
				},
				Value: &Identifier{
					Token: token.Token{Type: token.IDENT, Literal: "anotherVar"},
					Value: "anotherVar",
				},
			},
		},
	}

	expectation := "let myVar = anotherVar;"
	if program.String() != expectation {
		t.Errorf("program.String() wrong. expected='%s' received='%q'",
			expectation, program.String())
	}
}
