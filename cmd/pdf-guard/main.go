package main

import (
	"os"
	"pdfguard/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
