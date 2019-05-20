package main

import (
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"oradbauto/pkg/cmd"
	"os"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-oradb", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdOradb(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

