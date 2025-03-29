// Package dead implements the functionality exposed via the CLI.
package dead

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// Dead holds the configuration and state of the dead program.
type Dead struct {
	stdout io.Writer   // Normal program output is written here
	stderr io.Writer   // Logs, debug info and errors go here
	logger *log.Logger // The global logger
}

// New returns a new instance of [Dead].
func New(stdout, stderr io.Writer, debug bool) Dead {
	const width = 5

	level := log.InfoLevel
	if debug {
		level = log.DebugLevel
	}

	logger := log.NewWithOptions(stderr, log.Options{
		ReportTimestamp: true,
		Level:           level,
	})

	// Largely the default styles but with a default MaxWidth of 5 so as to not cutoff
	// DEBUG or ERROR
	logger.SetStyles(&log.Styles{
		Timestamp: lipgloss.NewStyle(),
		Caller:    lipgloss.NewStyle().Faint(true),
		Prefix:    lipgloss.NewStyle().Bold(true).Faint(true),
		Message:   lipgloss.NewStyle(),
		Key:       lipgloss.NewStyle().Faint(true),
		Value:     lipgloss.NewStyle(),
		Separator: lipgloss.NewStyle().Faint(true),
		Levels: map[log.Level]lipgloss.Style{
			log.DebugLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.DebugLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("63")),
			log.InfoLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.InfoLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("86")),
			log.WarnLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.WarnLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("192")),
			log.ErrorLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.ErrorLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("204")),
			log.FatalLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(log.FatalLevel.String())).
				Bold(true).
				MaxWidth(width).
				Foreground(lipgloss.Color("134")),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	})

	return Dead{stdout: stdout, stderr: stderr, logger: logger}
}

// CheckOptions are the flags passed to the check subcommand.
type CheckOptions struct {
	Debug bool // Enable verbose logging
}

// Check is the entry point for the `dead check` subcommand.
func (d Dead) Check(path string, options CheckOptions) error {
	d.logger.Debug("Checking links", "path", path)
	fmt.Fprintf(d.stdout, "Checking %s\n", path)
	return nil
}
