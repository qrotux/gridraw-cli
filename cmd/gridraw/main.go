package main

import (
	"os"

	"github.com/qrotux/gridraw-cli/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		cli.PrintError(os.Stderr, err)
		os.Exit(cli.ExitCode(err))
	}
}
