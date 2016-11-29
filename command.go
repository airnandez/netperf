package main

import (
	"flag"
)

type command struct {
	fset *flag.FlagSet
	run  func(args []string) error
}
