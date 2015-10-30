package main

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/docopt/docopt.go"
)

const (
	version = "v0.0.1"
	usage   = `usage:
	ersatz start <port> <definitions_dir>
	ersatz -h | --help
	ersatz --version

	options:
		-h --help  		show this screen
		--version  		show version
`
)

func main() {
	returnCode := entryPoint(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	os.Exit(returnCode)
}

func entryPoint(cliArgs []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	args, err := docopt.Parse(usage, cliArgs, true, version, true)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	if args["start"].(bool) {
		port, err := strconv.Atoi(args["<port>"].(string))
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		startApp := NewStartApp(port, args["<definitions_dir>"].(string))
		if err := startApp.Run(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	return 0
}