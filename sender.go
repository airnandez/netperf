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

type senderConfig struct {
	// Command line options
	help       bool
	addr       string
	duration   time.Duration
	parallel   int
	bufferSize string
}

func senderCmd() command {
	fset := flag.NewFlagSet("netperf send", flag.ExitOnError)
	config := senderConfig{}
	fset.BoolVar(&config.help, "help", false, "")
	fset.StringVar(&config.addr, "addr", defaultReceiverAddr, "")
	fset.DurationVar(&config.duration, "duration", defaultDuration, "")
	fset.IntVar(&config.parallel, "parallel", defaultParallel, "")
	fset.StringVar(&config.bufferSize, "len", defaultBufferSize, "")
	run := func(args []string) error {
		fset.Usage = func() {
			senderUsage(args[0], os.Stderr)
		}
		fset.Parse(args[1:])
		posArgs := fset.Args()
		if len(posArgs) != 0 {
			return fmt.Errorf("unexpected argument %q", posArgs[0])
		}
		return senderRun(args[0], config)
	}
	return command{fset: fset, run: run}
}

func senderRun(cmdName string, config senderConfig) error {
	if config.help {
		senderUsage(cmdName, os.Stderr)
		return nil
	}
	errlog = setErrlog(cmdName)
	bufsize, err := parseBufferLength(config.bufferSize)
	if err != nil {
		return fmt.Errorf("invalid buffer size value %q", config.bufferSize)
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
	dial := getDialer(config.addr)
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

	// Collect and print summary report
	report := <-summary
	if report.dataVolume > 0.0 {
		outlog.Printf("duration:                       %s\n", report.duration)
		outlog.Printf("streams:                        %d\n", report.numWorkers)
		outlog.Printf("data volume:                    %.2f MiB\n", report.dataVolume)
		outlog.Printf("aggregated throughput:          %.2f MiB/sec\n", report.aggregateThroughput)
		outlog.Printf("avg/std throughput per stream:  %.2f / %.2f MiB/sec\n", report.avgStreamThroughput, report.stdStreamThroughput)
	}
	if len(report.errors) > 0 {
		return report.errors[0]
	}
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
	dataVolume float64 // MiB
	throughput float64 // MiB/sec
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
	dataVolume          float64 // MiB
	aggregateThroughput float64 // MiB/sec
	avgStreamThroughput float64 // MiB/sec
	stdStreamThroughput float64 // MiB/sec
	duration            time.Duration
	errors              []error
}

func collectWorkerResponses(responses <-chan *workerResponse, summary chan<- summaryReport) {
	dataVolume := float64(0)
	numWorkers := 0
	start := time.Now().Add(3000 * time.Hour)
	end := time.Now().Add(-3000 * time.Hour)
	throughputs := make([]float64, 0, 128)
	errors := make([]error, 0, 128)
	for resp := range responses {
		numWorkers += 1
		if resp.start.Before(start) {
			start = resp.start
		}
		if resp.end.After(end) {
			end = resp.end
		}
		if resp.err != nil {
			errors = append(errors, resp.err)
			continue
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
		errors:              errors,
	}
}

// getDialer returns a function to dial to the server
// according to the format of the addr argument.
// addr can be of the form: 'host:port' or 'tls://host:port'.
func getDialer(addr string) func() (net.Conn, error) {
	const prefix = "tls://"
	d := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	if !strings.HasPrefix(addr, prefix) {
		return func() (net.Conn, error) {
			return d.Dial("tcp", addr)
		}
	}
	addr = strings.TrimPrefix(addr, prefix)
	config := tls.Config{
		InsecureSkipVerify: true,
	}
	return func() (net.Conn, error) {
		return tls.DialWithDialer(d, "tcp", addr, &config)
	}
}

func senderUsage(cmd string, f *os.File) {
	const template = `
USAGE:
{{.Tab1}}{{.AppName}} {{.SubCmd}} [-duration <duration>] [-len <buffer length>]
{{.Tab1}}{{.AppNameFiller}} {{.SubCmdFiller}} [-parallel <integer>] [-addr <network address>]
{{.Tab1}}{{.AppName}} {{.SubCmd}} -help

DESCRIPTION:
{{.Tab1}}'{{.AppName}} {{.SubCmd}}' establishes a network connection with the receiver
{{.Tab1}}for sending data to it. It reports the observed network throughput
{{.Tab1}}of that exchange.
{{.Tab1}}For this command to work, a receiver must be already running. To start a
{{.Tab1}}receiver use the command '{{.AppName}} {{.ReceiveSubCmd}}'

OPTIONS:
{{.Tab1}}-addr <network address>
{{.Tab2}}network address of the receiver. The form of the address is 'host:port'
{{.Tab2}}if the receiver expects a TCP connection, or 'tls://host:port'
{{.Tab2}}if the receiver expects a TLS connection.
{{.Tab2}}Default: '{{.DefaultReceiverAddr}}'

{{.Tab1}}-duration <duration>
{{.Tab2}}amount of time for sending data. Examples of valid values
{{.Tab2}}for this option are '60s', '1h30m', '120s', '2h', etc.
{{.Tab2}}Default: '{{.DefaultDuration}}'

{{.Tab1}}-len <buffer length>
{{.Tab2}}size in bytes of the buffer used for sending data to the receiver.
{{.Tab2}}Examples of valid values for this option are: '4096', '128K', '512KB',
{{.Tab2}}'1MB'. The suffix 'K' is understood to be 1024 and the suffix 'M' to be
{{.Tab2}}1024x1024.
{{.Tab2}}Default: '{{.DefaultBufferSize}}'

{{.Tab1}}-parallel <integer>
{{.Tab2}}number of simultaneous network connections to establish with the receiver.
{{.Tab2}}Default: {{.DefaultParallel}}

{{.Tab1}}-help
{{.Tab2}}print this help
`
	tmplFields["SubCmd"] = cmd
	tmplFields["SubCmdFiller"] = strings.Repeat(" ", len(cmd))
	tmplFields["DefaultReceiverAddr"] = defaultReceiverAddr
	tmplFields["ReceiveSubCmd"] = receiveSubCmd
	tmplFields["DefaultDuration"] = defaultDuration.String()
	tmplFields["DefaultBufferSize"] = defaultBufferSize
	tmplFields["DefaultParallel"] = fmt.Sprintf("%d", defaultParallel)
	render(template, tmplFields, f)
}
