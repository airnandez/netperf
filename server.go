package main

import (
	"flag"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

type serverConfig struct {
	// Command line options
	help bool
	addr string
}

func serverCmd() command {
	fset := flag.NewFlagSet("netperf server", flag.ExitOnError)
	config := serverConfig{}
	fset.BoolVar(&config.help, "help", false, "")
	fset.StringVar(&config.addr, "addr", defaultServerAddr, "")
	run := func(args []string) error {
		fset.Usage = func() {
			serverUsage(args[0], os.Stderr)
		}
		fset.Parse(args[1:])
		return serverRun(args[0], config)
	}
	return command{fset: fset, run: run}
}

func serverRun(cmdName string, config serverConfig) error {
	if config.help {
		serverUsage(cmdName, os.Stderr)
		return nil
	}
	errlog = setErrlog(cmdName)
	debug(1, "running server with:")
	debug(1, "   addr='%s'\n", config.addr)

	listener, err := net.Listen("tcp", config.addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			errlog.Printf("%s\n", err)
			continue
		}
		go handleConn(conn)
	}
	return nil
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	received := uint64(0)
	buffer := make([]byte, 256*1024)
	start := time.Now()
	for {
		n, err := conn.Read(buffer[:])
		if err != nil {
			if err == io.EOF {
				break
			}
			errlog.Printf("%s\n", err)
			return
		}
		received += uint64(n)
	}
	elapsed := time.Since(start)
	rate := float64(received) / float64(MB) / elapsed.Seconds()
	errlog.Printf("throughput: %.f MB/sec\n", rate)
}

func serverUsage(cmd string, f *os.File) {
	const template = `
USAGE:
{{.Tab1}}{{.AppName}} {{.SubCmd}} [-addr=<network address>]
{{.Tab1}}{{.AppName}} {{.SubCmd}} -help

DESCRIPTION:
{{.Tab1}}'{{.AppName}} {{.SubCmd}}' starts a server which waits for
{{.Tab1}}network connections from clients and start exchanging data with them.
{{.Tab1}}It reports on the network thoughput observed during the exchange.

OPTIONS:
{{.Tab1}}-addr=<network address>
{{.Tab2}}specifies the network address this server listens to for incoming
{{.Tab2}}connections. The form of each address is 'interface:port', for
{{.Tab2}}instance '127.0.0.1:9876'.
{{.Tab2}}Default: '{{.DefaultServerAddr}}'

{{.Tab1}}-help
{{.Tab2}}print this help
`
	tmplFields["SubCmd"] = cmd
	tmplFields["SubCmdFiller"] = strings.Repeat(" ", len(cmd))
	tmplFields["DefaultServerAddr"] = defaultServerAddr
	render(template, tmplFields, f)
}
