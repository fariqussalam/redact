package main

import (
	"os"

	"github.com/fariqussalam/redact/internal/cli"
)

func main() { os.Exit(cli.Main(os.Args[1:])) }
