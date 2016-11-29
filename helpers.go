package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

func setErrlog(cmd string) *log.Logger {
	return log.New(os.Stderr, fmt.Sprintf("%s %s: ", appName, cmd), log.Ldate|log.Ltime)
}

// parseBufferLength parses a string of any of the forms
//    1024
//    1024B
//    1024KB
//    1024K
//    1024M
//    1024MB
// and returns the equivalent number of bytes
func parseBufferLength(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.ToUpper(s)
	if s[len(s)-1] == 'B' {
		s = s[:len(s)-1]
	}
	factor := ByteSize(1)
	switch s[len(s)-1] {
	case 'K':
		s = s[:len(s)-1]
		factor = KB
	case 'M':
		s = s[:len(s)-1]
		factor = MB
	case 'G':
		s = s[:len(s)-1]
		factor = GB
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return v * int64(factor), nil
}

// stats returns the sum, average and standard deviation of a
// slice of floats
func stats(rates []float64) (sum, avg, std float64) {
	for _, r := range rates {
		sum += r
	}
	avg = sum / float64(len(rates))
	for _, r := range rates {
		std += (r - avg) * (r - avg)
	}
	std = math.Sqrt(std / float64(len(rates)))
	return
}
