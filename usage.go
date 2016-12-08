package main

import (
	"io"
	"os"
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
{{.Tab1}}{{.AppName}} {{.ReceiveCmd}} [options]
{{.Tab1}}{{.AppName}} {{.SendCmd}} [options]

{{.Tab1}}{{.AppName}} -help
{{.Tab1}}{{.AppName}} -version
{{if eq .UsageVersion "short"}}
Use '{{.AppName}} -help' to get more detailed usage information.{{else}}

DESCRIPTION:
{{.Tab1}}{{.AppName}} is a simple tool to measure throughput that can be obtained
{{.Tab1}}by a minimal Go application when transferring data over the network.

OPTIONS:
{{.Tab1}}-help
{{.Tab2}}Prints this help

{{.Tab1}}-version
{{.Tab2}}Show detailed version information about this application

SUBCOMMANDS:
{{.Tab1}}{{.ReceiveCmd}}
{{.Tab2}}use this subcommand to start a data receiver. A receiver waits
{{.Tab2}}for network connections from a sender, receives the data sent
{{.Tab2}}by it and reports on the observed throughput.

{{.Tab2}}Use '{{.AppName}} {{.ReceiveCmd}} -help' for getting detailed help on this
{{.Tab2}}subcommand.

{{.Tab1}}{{.SendCmd}}
{{.Tab2}}use this subcommand to establish a connection with a receiver,
{{.Tab2}}send data to it for a specified period of time and report on the
{{.Tab2}}observed throughtput.

{{.Tab2}}Use '{{.AppName}} {{.SendCmd}} -help' for getting detailed help on this
{{.Tab2}}subcommand.
{{end}}
`
	if kind == usageLong {
		tmplFields["UsageVersion"] = "long"
	}
	tmplFields["ReceiveCmd"] = receiveSubCmd
	tmplFields["SendCmd"] = sendSubCmd
	render(usageTempl, tmplFields, f)
}

func render(tpl string, fields map[string]string, out io.Writer) {
	minWidth, tabWidth, padding := 4, 4, 0
	tabwriter := tabwriter.NewWriter(out, minWidth, tabWidth, padding, byte(' '), 0)
	templ := template.Must(template.New("").Parse(tpl))
	templ.Execute(tabwriter, fields)
	tabwriter.Flush()
}
