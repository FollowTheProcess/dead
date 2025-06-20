// Package cmd implements dead's CLI.
package cmd

import (
	"runtime"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/dead/internal/dead"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

// Build returns the dead command line interface.
func Build() (*cli.Command, error) {
	return cli.New(
		"dead",
		cli.Short("A dead simple link checker"),
		cli.Version(version),
		cli.Commit(commit),
		cli.BuildDate(date),
		cli.SubCommands(check),
	)
}

// check returns the check subcommand.
func check() (*cli.Command, error) {
	var options dead.CheckOptions
	return cli.New(
		"check",
		cli.Short("Check a file or files in a directory (recursively) for dead links"),
		cli.RequiredArg("path", "Path to the file or directory to scan"),
		cli.Flag(&options.Debug, "debug", cli.NoShortHand, false, "Enable debug logging"),
		cli.Flag(&options.Timeout, "timeout", 't', dead.DefaultOverallTimeout, "Timeout for the entire operation"),
		cli.Flag(
			&options.RequestTimeout,
			"request-timeout",
			'r',
			dead.DefaultRequestTimeout,
			"Timeout for each request",
		),
		cli.Flag(
			&options.Parallelism,
			"parallelism",
			'p',
			runtime.NumCPU(),
			"Number of goroutines available for checking",
		),
		cli.Run(func(cmd *cli.Command, args []string) error {
			dead := dead.New(cmd.Stdout(), cmd.Stderr(), options.Debug, version)
			return dead.Check(cmd.Arg("path"), options)
		}),
	)
}
