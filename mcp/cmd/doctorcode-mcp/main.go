package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/maxqstudio/DoctorCode/internal/buildinfo"
	doctorcodemcp "github.com/maxqstudio/DoctorCode/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	flags := flag.NewFlagSet("doctorcode-mcp", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "repository root bound to this MCP server process")
	showVersion := flags.Bool("version", false, "print build version and exit")
	asJSON := flags.Bool("json", false, "print version metadata as JSON; valid only with --version")
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "doctorcode-mcp accepts no positional arguments")
		os.Exit(2)
	}
	if *asJSON && !*showVersion {
		fmt.Fprintln(os.Stderr, "--json is valid only with --version")
		os.Exit(2)
	}
	if *showVersion {
		info := buildinfo.Current()
		if *asJSON {
			if err := json.NewEncoder(os.Stdout).Encode(info); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("doctorcode-mcp %s commit=%s built=%s\n", info.Version, info.Commit, info.BuildDate)
		}
		return
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
