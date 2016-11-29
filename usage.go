package main

import (
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"
)

type usageType uint32

const (
	usageShort usageType = iota
	usageLong
)

func printUsage(f *os.File, kind usageType) {
	const usageTempl = `
USAGE:
{{.Tab1}}{{.AppName}} server [-addr=<network address>]

{{.Tab1}}{{.AppName}} client [-server=<network address>]

{{.Tab1}}{{.AppName}} -help
{{.Tab1}}{{.AppName}} -version
{{if eq .UsageVersion "short"}}
Use '{{.AppName}} -help' to get detailed information about options and examples
of usage.{{else}}

DESCRIPTION:
{{.Tab1}}{{.AppName}} is a basic tool to measure the network throughput that
{{.Tab1}}can be obtained by a Go application.

OPTIONS:
{{.Tab1}}-help
{{.Tab2}}Prints this help

{{.Tab1}}-version
{{.Tab2}}Show detailed version information about this application

SUBCOMMANDS:
{{.Tab1}}server
{{.Tab2}}use this subcommand to start a server. A server waits for network
{{.Tab2}}connections from a client and measures the thoughput of data
{{.Tab2}}exchanged between client and server.

{{.Tab2}}Use '{{.AppName}} server -help' for getting detailed help on this
{{.Tab2}}subcommand.

{{.Tab1}}client
{{.Tab2}}use this subcommand to connect to a server and start a data exchange
{{.Tab2}}with it and report on the observed throughtput.

{{.Tab2}}Use '{{.AppName}} client -help' for getting detailed help on this
{{.Tab2}}subcommand.
{{end}}
`
	tmplFields["ClientCmdFiller"] = strings.Repeat(" ", len("client"))
	tmplFields["ServerCmdFiller"] = strings.Repeat(" ", len("server"))
	if kind == usageLong {
		tmplFields["UsageVersion"] = "long"
	}
	render(usageTempl, tmplFields, f)
}

func render(tpl string, fields map[string]string, out io.Writer) {
	minWidth, tabWidth, padding := 4, 4, 0
	tabwriter := tabwriter.NewWriter(out, minWidth, tabWidth, padding, byte(' '), 0)
	templ := template.Must(template.New("").Parse(tpl))
	templ.Execute(tabwriter, fields)
	tabwriter.Flush()
}
