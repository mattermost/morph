package main

import (
	"fmt"
	"os"

	"github.com/go-morph/morph/commands"
)

func main() {
	if err := commands.RootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}
