package main

import (
	"os"

	"github.com/benedictjohannes/crobe/internal/runner"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	return runner.Run(args, runner.Options{
		Name:    "crobe",
		IsAgent: true,
	})
}

