package main

import (
	"github.com/kunit/cmdspy"
	"github.com/kunit/cmdspy/version"
	"os"
)

func main() {
	os.Exit(cmdspy.RunCLI(cmdspy.Env{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Args:    os.Args[1:],
		Version: version.Version,
	}))
}
