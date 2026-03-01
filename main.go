package main

import (
	_ "embed"
	"os"

	"github.com/matrixise/rmm-tracker/cmd"
)

//go:embed CHANGELOG.md
var changelogMD []byte

func init() {
	cmd.ChangelogMD = changelogMD
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
