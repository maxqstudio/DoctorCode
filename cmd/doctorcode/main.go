package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/scanner"
	"github.com/maxqstudio/DoctorCode/internal/toolchain"
)

const version = "0.0.1-dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "toolchains":
		runToolchains(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println(version)
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runScan(args []string) {
	root := "."
	asJSON := false
	for _, arg := range args {
		if arg == "--json" {
			asJSON = true
		} else if !strings.HasPrefix(arg, "-") {
			root = arg
		}
	}
	result, err := scanner.Scan(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		os.Exit(1)
	}
	if asJSON {
		writeJSON(result)
		return
	}
	fmt.Printf("ROOT %s\nFILES %d\nRECOGNIZED %d\n", result.Root, result.Files, result.RecognizedFiles)
	for _, item := range result.Languages {
		fmt.Printf("%s %d\n", item.Name, item.Files)
	}
}

func runToolchains(args []string) {
	root := "."
	asJSON := false
	for _, arg := range args {
		if arg == "--json" {
			asJSON = true
		} else if !strings.HasPrefix(arg, "-") {
			root = arg
		}
	}
	result := toolchain.Detect(root)
	if asJSON {
		writeJSON(result)
		return
	}
	for _, item := range result {
		state := "MISSING"
		if item.Available {
			state = "AVAILABLE"
		}
		fmt.Printf("%s %s %s", item.Language, state, item.Tool)
		if item.Path != "" {
			fmt.Printf(" %s", item.Path)
		}
		if len(item.Manifests) > 0 {
			fmt.Printf(" manifests=%s", strings.Join(item.Manifests, ","))
		}
		fmt.Println()
	}
}

func writeJSON(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "json encode failed:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`DoctorCode - deterministic code intelligence for small AI agents

Usage:
  doctorcode scan [path] [--json]
  doctorcode toolchains [path] [--json]
  doctorcode version

M00 bootstrap scope:
  - cross-platform repository inventory
  - language detection
  - compiler/toolchain capability detection

BLOAT, SECURITY, SIMPLIFY, LOGIC, and DEADCODE detectors are not yet claimed
as implemented until their dedicated acceptance evidence exists.`)
}
