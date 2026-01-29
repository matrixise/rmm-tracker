package main

import (
	"os"

	"github.com/matrixise/rmm-tracker/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
