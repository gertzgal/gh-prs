package main

import (
	"os"

	"github.com/gertzgal/gh-prs/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Environ()))
}
