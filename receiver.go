package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"
)

type receiverConfig struct {
	// Command line options
	help bool
	addr string
	ca   string
	cert string
	key  string
}

func receiverCmd() command {
	fset := flag.NewFlagSet("netperf receive", flag.ExitOnError)
	config := receiverConfig{}
	fset.BoolVar(&config.help, "help", false, "")
	fset.StringVar(&config.addr, "addr", defaultReceiverAddr, "")
	fset.StringVar(&config.ca, "ca", defaultReceiverCA, "")
	fset.StringVar(&config.cert, "cert", defaultReceiverCert, "")
	fset.StringVar(&config.key, "key", defaultReceiverKey, "")
	run := func(args []string) error {
		fset.Usage = func() {
			receiverUsage(args[0], os.Stderr)
		}
		fset.Parse(args[1:])
		posArgs := fset.Args()
		if len(posArgs) != 0 {
			return fmt.Errorf("unexpected argument %q", posArgs[0])
		}
		return receiverRun(args[0], config)
	}
	return command{fset: fset, run: run}
}

func receiverRun(cmdName string, config receiverConfig) error {
	if config.help {
		receiverUsage(cmdName, os.Stderr)
		return nil
	}
	errlog = setErrlog(cmdName)
	listener, err := listen(config)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			errlog.Printf("%s\n", err)
			continue
		}
		go receiveData(conn)
	}
	return nil
}

func listen(config receiverConfig) (net.Listener, error) {
	const prefix = "tls://"
	if !strings.HasPrefix(config.addr, prefix) {
		return net.Listen("tcp", config.addr)
	}
	pool, err := loadCaCerts(config.ca)
	if err != nil {
		return nil, fmt.Errorf("error loading CA certificate from file %q: %s", config.ca, err)
	}
	serverCert, err := tls.LoadX509KeyPair(config.cert, config.key)
	if err != nil {
		return nil, fmt.Errorf("error loading server certificate from files %q and %q: %s", config.cert, config.key, err)
	}
	tlsConfig := &tls.Config{
		ClientCAs:    pool,
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}
	config.addr = strings.TrimPrefix(config.addr, prefix)
	return tls.Listen("tcp", config.addr, tlsConfig)
}

func loadCaCerts(path string) (*x509.CertPool, error) {
	if path == "" {
		return nil, nil
	}
	caCerts, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCerts) {
		return nil, err
	}
	return pool, nil
}

func receiveData(conn net.Conn) {
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
	errlog.Printf("throughput: %.2f MiB/sec\n", rate)
}

func receiverUsage(cmd string, f *os.File) {
	const template = `
USAGE:
{{.Tab1}}{{.AppName}} {{.SubCmd}} [-ca <file>] [-cert <file>] [-key <file>]
{{.Tab1}}{{.AppNameFiller}} {{.SubCmdFiller}} [-addr <network address>]
{{.Tab1}}{{.AppName}} {{.SubCmd}} -help

DESCRIPTION:
{{.Tab1}}'{{.AppName}} {{.SubCmd}}' starts a receiver which waits for incoming
{{.Tab1}}network connections from senders, receives and discards data from them.
{{.Tab1}}It reports on the network thoughput observed while receiving the data.

OPTIONS:
{{.Tab1}}-addr <network address>
{{.Tab2}}specifies the network address this receiver listens to for incoming
{{.Tab2}}connections. The form of this address is 'interface:port' or
{{.Tab2}}'tls://interface:port'. Examples of valid adresses are '127.0.0.1:9876'
{{.Tab2}}'tls://127.0.0.1:9876'.
{{.Tab2}}Use a network address starting by 'tls://' to instruct the server to
{{.Tab2}}use TLS to encrypt the communication channel with senders.
{{.Tab2}}Default: '{{.DefaultReceiverAddr}}'

{{.Tab1}}-cert <file>
{{.Tab2}}specifies the path of the PEM-formatted file of the certificate
{{.Tab2}}this receiver presents to its clients when using TLS connections.
{{.Tab2}}Default: '{{.DefaultReceiverCert}}'

{{.Tab1}}-key <file>
{{.Tab2}}specifies the path of the PEM-formatted file of the private key
{{.Tab2}}this receiver uses to encrypt the communication channel with its
{{.Tab2}}clients, when using TLS.
{{.Tab2}}Default: '{{.DefaultReceiverKey}}'

{{.Tab1}}-ca <file>
{{.Tab2}}specifies the path of the PEM-formatted file of CA certificates.
{{.Tab2}}This server accepts client certificates issued by any of those CAs.
{{.Tab2}}This option is only relevant when using TLS.
{{.Tab2}}Default: '{{.DefaultReceiverCA}}'

{{.Tab1}}-help
{{.Tab2}}print this help
`
	tmplFields["SubCmd"] = cmd
	tmplFields["SubCmdFiller"] = strings.Repeat(" ", len(cmd))
	tmplFields["DefaultReceiverAddr"] = defaultReceiverAddr
	tmplFields["DefaultReceiverCert"] = defaultReceiverCert
	tmplFields["DefaultReceiverKey"] = defaultReceiverKey
	tmplFields["DefaultReceiverCA"] = defaultReceiverCA
	render(template, tmplFields, f)
}
