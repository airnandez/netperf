package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strconv"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	logFlags = log.Ldate | log.Ltime
)

var (
	debugLogger = log.New(os.Stderr, "", logFlags)
	debugLevel  = 0 // set to 0 (zero) for disabling debug
	prefixFmt   = ""
	formatMap   = map[bool]string{
		true:  "\033[33mDEBUG L%d [%s:%d]\033[0m \t",
		false: "DEBUG L%d [%s:%d] \t",
	}
)

func init() {
	// Determine if output format to use according to the destination of the
	// debug messages. If stderr is a terminal we can use colors
	prefixFmt = formatMap[terminal.IsTerminal(int(os.Stderr.Fd()))]

	// Set the debug level from the environmental variable
	if env := os.Getenv("NETPERF_DEBUG"); env != "" {
		if value, err := strconv.ParseInt(env, 10, 32); err == nil {
			debugLevel = clamp(int(value), 0, 5)
		}
	}
}

func setDebugLevel(level int) {
	debugLevel = level
}

func isDebugActive() bool {
	return debugLevel > 0
}

// Show a debug message
func debug(level int, format string, v ...interface{}) {
	if debugLevel > 0 && level <= debugLevel {
		_, file, line, _ := runtime.Caller(1)
		debugLogger.SetPrefix(fmt.Sprintf(prefixFmt, level, path.Base(file), line))
		debugLogger.Printf(format, v...)
	}
}

// Returns the minimum value among two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Returns the maximum value among two integers
func maxInt(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// Keeps a given value within the specified interval
func clamp(val, min, max int) int {
	return minInt(maxInt(min, val), max)
}
