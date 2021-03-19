package main

import (
	"os"

	"github.com/go-morph/morph"

	"github.com/go-morph/morph/commands"
)

func main() {
	if err := commands.RootCmd().Execute(); err != nil {
		morph.ErrorLogger.Fprintf(os.Stderr, "An Error Occurred\n")
		_, _ = morph.ErrorLoggerLight.Fprintf(os.Stderr, "--> %v\n", err)
	}
}
