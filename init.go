package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	// application name
	appName string

	// application version: set at build time based on git tag
	appVersion = "dev"

	// application build time stamp: set at build time
	appBuildTime = "unknown"

	// application output and error logger
	errlog *log.Logger
	outlog *log.Logger

	// fields used in the help templates
	tmplFields = map[string]string{
		"Sp2":          "  ",
		"Sp3":          "   ",
		"Sp4":          "    ",
		"Sp5":          "     ",
		"Sp6":          "      ",
		"Sp7":          "       ",
		"Sp8":          "        ",
		"Tab1":         "\t",
		"Tab2":         "\t\t",
		"Tab3":         "\t\t\t",
		"Tab4":         "\t\t\t\t",
		"Tab5":         "\t\t\t\t\t",
		"Tab6":         "\t\t\t\t\t\t",
		"UsageVersion": "short",
		"AppVersion":   appVersion,
		"BuildTime":    appBuildTime,
		"Os":           runtime.GOOS,
		"Arch":         runtime.GOARCH,
		"GoVersion":    runtime.Version(),
	}
)

const (
	receiveSubCmd       string        = "receive"
	sendSubCmd          string        = "send"
	defaultReceiverAddr string        = ":9876"
	defaultReceiverCA   string        = "ca.pem"
	defaultReceiverCert string        = "cert.pem"
	defaultReceiverKey  string        = "key.pem"
	defaultDuration     time.Duration = time.Duration(30) * time.Second
	defaultParallel     int           = 1
	defaultBufferSize   string        = "128KB"
)

func init() {
	appName = filepath.Base(os.Args[0])
	errlog = log.New(os.Stderr, fmt.Sprintf("%s: ", appName), 0)
	outlog = log.New(os.Stdout, fmt.Sprintf("%s: ", appName), 0)
	tmplFields["AppName"] = appName
	tmplFields["AppNameFiller"] = strings.Repeat(" ", len(appName))
	return
}

type ByteSize int64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
)
