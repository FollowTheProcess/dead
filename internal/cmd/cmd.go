// Package cmd implements dead's CLI.
package cmd

import (
	"context"
	"runtime"

	"go.followtheprocess.codes/cli"
	"go.followtheprocess.codes/cli/flag"
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
		cli.Arg(&options.Path, "path", "Path to the file or directory to scan"),
		cli.Flag(&options.Debug, "debug", flag.NoShortHand, "Enable debug logging"),
		cli.Flag(
			&options.Timeout,
			"timeout",
			't',
			"Timeout for the entire operation",
			cli.FlagDefault(dead.DefaultOverallTimeout),
		),
		cli.Flag(
			&options.RequestTimeout,
			"request-timeout",
			'r',
			"Timeout for each request",
			cli.FlagDefault(dead.DefaultRequestTimeout),
		),
		cli.Flag(
			&options.Parallelism,
			"parallelism",
			'p',
			"Number of goroutines available for checking",
			cli.FlagDefault(runtime.NumCPU()),
		),
		cli.Run(func(ctx context.Context, cmd *cli.Command) error {
			dead := dead.New(cmd.Stdout(), cmd.Stderr(), options.Debug, version)
			return dead.Check(ctx, options)
		}),
	)
}
