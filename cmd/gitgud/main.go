// Command gitgud is a small GitFlow-style workflow layer over git.
package main

import (
	"os"

	"github.com/josepalala-dev/gitgud/internal/cli"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		os.Stderr.WriteString("gitgud: " + err.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(cli.Run(wd, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
