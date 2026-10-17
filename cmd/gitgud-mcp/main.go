// Command gitgud-mcp serves gitgud's commands as MCP tools over stdio, so an
// MCP client such as Claude Code can call them. Stdout carries the protocol;
// everything else goes to stderr.
package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/josepalala-dev/gitgud/internal/cli"
	"github.com/josepalala-dev/gitgud/internal/mcpserver"
)

func main() {
	s := mcpserver.New(cli.Version, mcpserver.ExecRunner)
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, "gitgud-mcp:", err)
		os.Exit(1)
	}
}
