package main

import (
	"os"

	"github.com/fariq/redact/internal/cli"
)

func main() { os.Exit(cli.Main(os.Args[1:])) }
