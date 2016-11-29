package main

import (
	"flag"
	"os"
)

func main() {
	fset := flag.NewFlagSet("netperf", flag.ExitOnError)
	fset.Usage = func() {
		printUsage(os.Stderr, usageShort)
	}
	version := fset.Bool("version", false, "print version number")
	help := fset.Bool("help", false, "print help")
	fset.Parse(os.Args[1:])

	if *version {
		printVersion(os.Stderr)
		os.Exit(0)
	}
	if *help {
		printUsage(os.Stderr, usageLong)
		os.Exit(0)
	}
	args := fset.Args()
	if len(args) == 0 {
		fset.Usage()
		return
	}

	commands := map[string]command{
		serverSubCmd: serverCmd(),
		clientSubCmd: clientCmd(),
	}
	cmd, ok := commands[args[0]]
	if !ok {
		errlog.Printf("'%s' is not an accepted command\n", args[0])
		fset.Usage()
		os.Exit(1)
	}
	if err := cmd.run(args); err != nil {
		errlog.Printf("%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
