package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	doctorcodemcp "github.com/maxqstudio/DoctorCode/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	flags := flag.NewFlagSet("doctorcode-mcp", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "repository root bound to this MCP server process")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "doctorcode-mcp accepts no positional arguments")
		os.Exit(2)
	}

	server, err := doctorcodemcp.NewServer(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
