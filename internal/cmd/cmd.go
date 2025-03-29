// Package cmd implements dead's CLI.
package cmd

import (
	"github.com/FollowTheProcess/cli"
	"github.com/FollowTheProcess/dead/internal/dead"
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
		cli.Run(func(cmd *cli.Command, args []string) error {
			dead := dead.New(cmd.Stdout(), cmd.Stderr(), options.Debug)
			return dead.Check(cmd.Arg("path"), options)
		}),
	)
}
