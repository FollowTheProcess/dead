// Package dead implements the functionality exposed via the CLI.
package dead

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/FollowTheProcess/dead/internal/extractor"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

const (
	// DefaultRequestTimeout is the default value for the per-request timeout.
	DefaultRequestTimeout = 5 * time.Second

	// for checking all detected links.
	DefaultOverallTimeout = 30 * time.Second
)

// TabWriter config.
const (
	minWidth = 1   // Min cell width
	tabWidth = 8   // Tab width in spaces
	padding  = 2   // Padding
	padChar  = ' ' // Char to pad with
	flags    = 0   // Config flags
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
	Debug          bool          // Enable verbose logging
	RequestTimeout time.Duration // Per request timeout
	Timeout        time.Duration // Timeout for the whole operation
}

// CheckResult holds the result of a link check.
type CheckResult struct {
	Err        error
	URL        string
	Status     string
	StatusCode int
	Duration   time.Duration
}

// Check is the entry point for the `dead check` subcommand.
func (d Dead) Check(path string, options CheckOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	// See if path is a file or a directory
	// File: Detect the type and use the right extractor, then extract the links
	// Dir: Traverse it recursively, extract links from all files until we've hit the bottom
	// probs concurrently in a pipeline across goroutines

	// The we have a load of links, fan them out over a load of goroutines in a pipeline
	// doing a http get in each

	// Range over the results channel which should return the URL, status code and the message
	// if it's not ok. But not stop the loop, we need to keep processing all links
	d.logger.Debug("Checking links", "path", path)

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("os.Stat(%s): %w", path, err)
	}

	if info.Mode().IsDir() {
		return errors.New("TODO: Support recursing a directory searching for links")
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s has unsupported file mode: %s", path, info.Mode())
	}

	// Now we know it's a regular file
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// TODO(@FollowTheProcess): This is all synchronous while I polish it and ensure it works
	// once I have good test coverage and the UX I want, let's make it fan out in a pipeline

	ext := extractor.NewText(f)
	links, err := ext.Extract()
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: options.RequestTimeout,
	}

	requests := make([]*http.Request, 0, len(links))
	for _, link := range links {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
		if err != nil {
			return err
		}
		// TODO(@FollowTheProcess): Set User-Agent
		requests = append(requests, request)
	}

	results := make([]CheckResult, 0, len(links))
	for _, request := range requests {
		var result CheckResult
		start := time.Now()
		response, err := client.Do(request)
		if err != nil {
			result = CheckResult{
				URL:      request.URL.String(),
				Err:      err,
				Duration: time.Since(start),
			}
		} else {
			result = CheckResult{
				URL:        request.URL.String(),
				StatusCode: response.StatusCode,
				Status:     response.Status,
				Err:        err,
				Duration:   time.Since(start),
			}
			response.Body.Close()
		}

		results = append(results, result)
	}

	tw := tabwriter.NewWriter(d.stdout, minWidth, tabWidth, padding, padChar, flags)
	for _, result := range results {
		if result.Err != nil {
			fmt.Fprintf(tw, "%s\t%v\t%s\n", result.URL, result.Err, result.Duration)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", result.URL, result.Status, result.Duration)
		}
	}

	tw.Flush()

	return nil
}
