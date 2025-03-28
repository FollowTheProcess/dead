// Package dead implements the functionality exposed via the CLI.
package dead

import (
	"fmt"
	"io"
)

// Dead holds the configuration and state of the dead program.
type Dead struct {
	stdout io.Writer // Normal program output is written here
	stderr io.Writer // Logs, debug info and errors go here
}

// New returns a new instance of [Dead].
func New(stdout, stderr io.Writer) Dead {
	return Dead{stdout: stdout, stderr: stderr}
}

// Check is the entry point for the `dead check` subcommand.
func (d Dead) Check(path string) error {
	fmt.Fprintf(d.stdout, "Checking %s\n", path)
	return nil
}
