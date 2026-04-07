package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mvanhorn/cli-printing-press/internal/version"
)

func main() {
	s := server.NewMCPServer("printing-press-mcp", version.Version,
		server.WithToolCapabilities(true)) // true enables tools/list_changed

	// TODO: Units 4-6 wire up registry fetch, handlers, meta-tools

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}
