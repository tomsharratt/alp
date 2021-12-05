package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/tomsharratt/alp/repl"
)

func main() {
	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Hello %s! This is the Alp programming language!\n",
		user.Username)
	fmt.Printf("REPL is ready for work...\n")
	repl.Run(os.Stdin, os.Stdout)
}
