package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type clientConfig struct {
	// Command line options
	help       bool
	serverAddr string
	duration   time.Duration
	parallel   int
	bufferSize string
}

func clientCmd() command {
	fset := flag.NewFlagSet("netperf client", flag.ExitOnError)
	config := clientConfig{}
	fset.BoolVar(&config.help, "help", false, "")
	fset.StringVar(&config.serverAddr, "server", defaultServerAddr, "")
	fset.DurationVar(&config.duration, "duration", defaultDuration, "")
	fset.IntVar(&config.parallel, "parallel", defaultParallel, "")
	fset.StringVar(&config.bufferSize, "len", defaultBufferSize, "")
	run := func(args []string) error {
		fset.Usage = func() {
			clientUsage(args[0], os.Stderr)
		}
		fset.Parse(args[1:])
		posArgs := fset.Args()
		if len(posArgs) != 0 {
			return fmt.Errorf("unexpected argument %q", posArgs[0])
		}
		return clientRun(args[0], config)
	}
	return command{fset: fset, run: run}
}

func clientRun(cmdName string, config clientConfig) error {
	if config.help {
		clientUsage(cmdName, os.Stderr)
		return nil
	}
	errlog = setErrlog(cmdName)
	debug(1, "running client with:")
	debug(1, "   serverAddr='%s'\n", config.serverAddr)
	debug(1, "   duration='%s'\n", config.duration)
	debug(1, "   parallel=%d\n", config.parallel)
	debug(1, "   bufferSize=%s\n", config.bufferSize)

	bufsize, err := parseBufferLength(config.bufferSize)
	if err != nil {
		return fmt.Errorf("invalid buffer size value '%s'", config.bufferSize)
	}

	// Start workers
	numWorkers := config.parallel
	if numWorkers <= 0 {
		numWorkers = 1
	}
	requests := make(chan *workerRequest, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i, &wg, requests)
	}

	// Establish connections to server, one per worker
	conns := make([]net.Conn, numWorkers)
	dial := getDialer(config.serverAddr)
	for i := 0; i < numWorkers; i++ {
		conn, err := dial()
		if err != nil {
			return err
		}
		conns[i] = conn
	}

	// Collect responses from workers
	responses := make(chan *workerResponse, numWorkers)
	summary := make(chan summaryReport)
	go collectWorkerResponses(responses, summary)

	// Submit requests to workers
	buffer := make([]byte, bufsize)
	for _, conn := range conns {
		requests <- &workerRequest{
			conn:     conn,
			buffer:   buffer,
			duration: config.duration,
			replyTo:  responses,
		}
	}
	close(requests)

	// Wait for workers to finish their execution
	wg.Wait()
	close(responses)

	// Close network connections
	for _, conn := range conns {
		conn.Close()
	}

	// Collect summary report
	report := <-summary
	outlog.Printf("duration:               %s\n", report.duration)
	outlog.Printf("streams:                %d\n", report.numWorkers)
	outlog.Printf("data volume:            %.f MB\n", report.dataVolume)
	outlog.Printf("aggregated throughput:  %.f MB/sec\n", report.aggregateThroughput)
	outlog.Printf("per stream throughput:  %.f ± %.f MB/sec\n", report.avgStreamThroughput, report.stdStreamThroughput)
	return nil
}

type workerRequest struct {
	conn     net.Conn
	duration time.Duration
	buffer   []byte
	replyTo  chan *workerResponse
}

type workerResponse struct {
	req        *workerRequest
	err        error
	start      time.Time
	end        time.Time
	dataVolume float64 // MB
	throughput float64 // MB/sec
}

func worker(workerID int, wg *sync.WaitGroup, requests <-chan *workerRequest) {
	defer wg.Done()
	for req := range requests {
		sent := float64(0)
		resp := &workerResponse{
			req:   req,
			start: time.Now(),
		}
		timeout := time.After(req.duration)
	loop:
		for {
			select {
			case <-timeout:
				// Stop sending data
				break loop

			default:
				n, err := req.conn.Write(req.buffer)
				sent += float64(n)
				if err != nil {
					resp.err = err
					break loop
				}
			}
		}
		if resp.err == nil {
			resp.end = time.Now()
			resp.dataVolume = sent / float64(MB)
			resp.throughput = resp.dataVolume / resp.end.Sub(resp.start).Seconds()
		}
		req.replyTo <- resp
	}
}

type summaryReport struct {
	numWorkers          int
	dataVolume          float64 // MB
	aggregateThroughput float64 // MB/sec
	avgStreamThroughput float64 // MB/sec
	stdStreamThroughput float64 // MB/sec
	duration            time.Duration
}

func collectWorkerResponses(responses <-chan *workerResponse, summary chan<- summaryReport) {
	var dataVolume float64
	numWorkers := 0
	start := time.Now().Add(3000 * time.Hour)
	end := time.Now().Add(-3000 * time.Hour)
	throughputs := make([]float64, 0, 128)
	for resp := range responses {
		numWorkers += 1
		if resp.err != nil {
			errlog.Printf("ERROR: %s\n", resp.err)
			continue
		}
		if resp.start.Before(start) {
			start = resp.start
		}
		if resp.end.After(end) {
			end = resp.end
		}
		dataVolume += resp.dataVolume
		throughputs = append(throughputs, resp.throughput)
	}
	_, avg, std := stats(throughputs)
	summary <- summaryReport{
		numWorkers:          numWorkers,
		dataVolume:          dataVolume,
		aggregateThroughput: dataVolume / end.Sub(start).Seconds(),
		avgStreamThroughput: avg,
		stdStreamThroughput: std,
		duration:            end.Sub(start),
	}
}

// getDialer returns a function to dial to the server
// according to the format of the addr argument
func getDialer(addr string) func() (net.Conn, error) {
	const prefix = "tls://"
	if !strings.HasPrefix(addr, prefix) {
		return func() (net.Conn, error) {
			return net.Dial("tcp", addr)
		}
	}
	addr = strings.TrimPrefix(addr, prefix)
	config := tls.Config{
		InsecureSkipVerify: true,
	}
	return func() (net.Conn, error) {
		return tls.Dial("tcp", addr, &config)
	}
}

func clientUsage(cmd string, f *os.File) {
	const template = `
USAGE:
{{.Tab1}}{{.AppName}} {{.SubCmd}} [-duration <duration>] [-len <buffer length>]
{{.Tab1}}{{.AppNameFiller}} {{.SubCmdFiller}} [-parallel <integer>] [-server <network address>]
{{.Tab1}}{{.AppName}} {{.SubCmd}} -help

DESCRIPTION:
{{.Tab1}}'{{.AppName}} {{.SubCmd}}' establishes a network connection with the server
{{.Tab1}}for sending data. It reports the observed network throughput
{{.Tab1}}of that exchange.
{{.Tab1}}For this command to work, a server must be already running. To start a
{{.Tab1}}server uset the command '{{.AppName}} {{.ServerSubCmd}}'

OPTIONS:
{{.Tab1}}-server <network address>
{{.Tab2}}network address of the server. The form of the address is 'host:port'
{{.Tab2}}if the server waits for a TCP connection, or 'tls://host:port'
{{.Tab2}}if the server waits for a TLS connection.
{{.Tab2}}Default: '{{.DefaultServerAddr}}'

{{.Tab1}}-duration <duration>
{{.Tab2}}amount of time of the data exchange. Examples of valid values
{{.Tab2}}for this option are '60s', '1h30m', '120s', '2h', etc.
{{.Tab2}}Default: '{{.DefaultDuration}}'

{{.Tab1}}-len <buffer length>
{{.Tab2}}size in bytes of the buffer used for sending data to the server.
{{.Tab2}}Examples of valid values for this option are: '4096', '128K', '512KB',
{{.Tab2}}'1MB'.
{{.Tab2}}Default: '{{.DefaultBufferSize}}'

{{.Tab1}}-parallel <integer>
{{.Tab2}}number of simultaneous network connections to establish with the server.
{{.Tab2}}Default: {{.DefaultParallel}}

{{.Tab1}}-help
{{.Tab2}}print this help
`
	tmplFields["SubCmd"] = cmd
	tmplFields["SubCmdFiller"] = strings.Repeat(" ", len(cmd))
	tmplFields["DefaultServerAddr"] = defaultServerAddr
	tmplFields["ServerSubCmd"] = serverSubCmd
	tmplFields["DefaultDuration"] = defaultDuration.String()
	tmplFields["DefaultBufferSize"] = defaultBufferSize
	tmplFields["DefaultParallel"] = fmt.Sprintf("%d", defaultParallel)
	render(template, tmplFields, f)
}
