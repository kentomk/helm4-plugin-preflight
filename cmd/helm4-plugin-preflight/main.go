package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kentomk/helm4-plugin-preflight/internal/analyzer"
)

var version = "0.1.0-dev"

type repeatedStrings []string

func (values *repeatedStrings) String() string {
	return fmt.Sprint([]string(*values))
}

func (values *repeatedStrings) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		writeUsage(stderr)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		writeUsage(stdout)
		return 0
	case "version":
		fmt.Fprintf(stdout, "helm4-plugin-preflight %s\n", version)
		return 0
	case "check":
		return runCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		return 2
	}
}

func writeUsage(out io.Writer) {
	fmt.Fprintln(out, `helm4-plugin-preflight checks repositories for Helm 4 plugin migration hazards.

Usage:
  helm4-plugin-preflight check [--root PATH] [--helm-plugins PATH] [--shell-file PATH ...] [--format text|json|sarif]
  helm4-plugin-preflight version
  helm4-plugin-preflight help

Run "helm4-plugin-preflight check --help" for check options.`)
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	helmPlugins := flags.String("helm-plugins", "", "installed Helm plugins directory (optional)")
	var shellFiles repeatedStrings
	flags.Var(&shellFiles, "shell-file", "repository-root-confined shell file to scan (repeatable)")
	format := flags.String("format", "text", "output format: text, json, or sarif")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "check does not accept positional arguments")
		return 2
	}
	if *format != "text" && *format != "json" && *format != "sarif" {
		fmt.Fprintf(stderr, "unsupported format %q (expected text, json, or sarif)\n", *format)
		return 2
	}

	report, err := analyzer.Check(*root, *helmPlugins, shellFiles, version)
	if err != nil {
		fmt.Fprintf(stderr, "check failed: %v\n", err)
		return 2
	}
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "write JSON: %v\n", err)
			return 2
		}
	} else if *format == "sarif" {
		if err := writeSARIF(stdout, report); err != nil {
			fmt.Fprintf(stderr, "write SARIF: %v\n", err)
			return 2
		}
	} else {
		writeText(stdout, report)
	}
	if report.Summary.Errors > 0 || report.Summary.Warnings > 0 {
		return 1
	}
	return 0
}

func writeText(out io.Writer, report analyzer.Report) {
	for _, diagnostic := range report.Diagnostics {
		fmt.Fprintf(out, "%s:%d:%d: %s %s %s\n", diagnostic.Path, diagnostic.Line, diagnostic.Column, diagnostic.Severity, diagnostic.RuleID, diagnostic.Message)
	}
	if len(report.Diagnostics) == 0 {
		fmt.Fprintf(out, "No Helm 4 plugin migration findings in %d input file(s).\n", report.Summary.FilesScanned)
		return
	}
	fmt.Fprintf(out, "%d finding(s) in %d input file(s).\n", len(report.Diagnostics), report.Summary.FilesScanned)
}
