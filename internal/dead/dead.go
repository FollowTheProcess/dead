// Package dead implements the functionality exposed via the CLI.
package dead

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/FollowTheProcess/dead/internal/extractor"
	"github.com/FollowTheProcess/hue"
	"github.com/FollowTheProcess/hue/tabwriter"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

const (
	// DefaultRequestTimeout is the default value for the per-request timeout.
	DefaultRequestTimeout = 5 * time.Second

	// DefaultOverallTimeout is the default timeout for checking all detected links.
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

// Hue styles.
const (
	success  = hue.Green | hue.Bold
	failure  = hue.Red | hue.Bold
	duration = hue.BrightBlack
)

// Dead holds the configuration and state of the dead program.
type Dead struct {
	stdout  io.Writer
	stderr  io.Writer
	logger  *log.Logger
	version string
}

// New returns a new instance of [Dead].
func New(stdout, stderr io.Writer, debug bool, version string) Dead {
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

	return Dead{
		stdout:  stdout,
		stderr:  stderr,
		logger:  logger,
		version: version,
	}
}

// CheckOptions are the flags passed to the check subcommand.
type CheckOptions struct {
	Debug          bool          // Enable verbose logging
	RequestTimeout time.Duration // Per request timeout
	Timeout        time.Duration // Timeout for the whole operation
	Parallelism    int           // Number of goroutines available for link checking
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
	if options.Parallelism < 1 {
		options.Parallelism = runtime.NumCPU()
	}
	start := time.Now()

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
	logger := d.logger.With("path", path)
	logger.Debug("Checking links")

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

	client := &http.Client{
		Timeout: options.RequestTimeout,
	}

	links := extractLinks(ctx, f)

	workers := make([]<-chan CheckResult, 0, options.Parallelism)
	for range options.Parallelism {
		workers = append(workers, check(ctx, client, links))
	}
	results := collect(ctx, workers...)

	var sorted []CheckResult //nolint:prealloc // It's channels so we don't actually know
	for result := range results {
		sorted = append(sorted, result)
	}

	// Sort results by URL on the way out so output is deterministic
	slices.SortFunc(sorted, func(a, b CheckResult) int {
		return cmp.Compare(a.URL, b.URL)
	})

	tw := tabwriter.NewWriter(d.stdout, minWidth, tabWidth, padding, padChar, flags)
	for _, result := range sorted {
		if result.Err != nil {
			fmt.Fprintf(
				tw,
				"%s\t%v\t%s\n",
				result.URL,
				failure.Text(result.Err.Error()),
				duration.Text(result.Duration.String()),
			)
			continue
		}

		if result.StatusCode >= http.StatusBadRequest {
			fmt.Fprintf(
				tw,
				"%s\t%s\t%s\n",
				result.URL,
				failure.Text(result.Status),
				duration.Text(result.Duration.String()),
			)
			continue
		}

		fmt.Fprintf(
			tw,
			"%s\t%s\t%s\n",
			result.URL,
			success.Text(result.Status),
			duration.Text(result.Duration.String()),
		)
	}

	tw.Flush()

	duration.Fprintf(
		d.stdout,
		"\nChecked %d links in %s (%d workers)\n",
		len(sorted),
		time.Since(start),
		options.Parallelism,
	)
	return nil
}

// result is a generic result type.
type result[T any] struct {
	value T
	err   error
}

// extractLinks extracts URLs from a reader and puts them on a channel to
// be processed.
func extractLinks(ctx context.Context, r io.Reader) <-chan result[string] {
	results := make(chan result[string])
	go func() {
		defer close(results)

		ext := extractor.NewText(r)
		links, err := ext.Extract()
		if err != nil {
			results <- result[string]{err: err}
			return
		}

		for _, link := range links {
			select {
			case <-ctx.Done():
				return
			default:
				results <- result[string]{value: link}
			}
		}
	}()

	return results
}

// check takes a channel of URLs to check and puts the results on the output channel.
func check(ctx context.Context, client *http.Client, in <-chan result[string]) <-chan CheckResult {
	results := make(chan CheckResult)
	go func() {
		defer close(results)
		for link := range in {
			select {
			case <-ctx.Done():
				return
			default:
				if link.err != nil {
					results <- CheckResult{URL: link.value, Err: link.err}
					return
				}
				request, err := http.NewRequestWithContext(ctx, http.MethodGet, link.value, nil)
				if err != nil {
					results <- CheckResult{URL: link.value, Err: err}
					return
				}
				request.Header.Add("User-Agent", "github.com/FollowTheProcess/dead")

				var result CheckResult
				requestStart := time.Now()
				response, err := client.Do(request)
				if err != nil {
					result = CheckResult{
						URL:      request.URL.String(),
						Err:      err,
						Duration: time.Since(requestStart),
					}
				} else {
					result = CheckResult{
						URL:        request.URL.String(),
						StatusCode: response.StatusCode,
						Status:     response.Status,
						Err:        err,
						Duration:   time.Since(requestStart),
					}
					response.Body.Close()
				}

				results <- result
			}
		}
	}()

	return results
}

// collect multiplexes a number of channels onto a single results channel.
//
// It is the fan-in side of the pipeline.
func collect(ctx context.Context, channels ...<-chan CheckResult) <-chan CheckResult {
	var wg sync.WaitGroup
	combined := make(chan CheckResult)

	wg.Add(len(channels))
	for _, channel := range channels {
		go func() {
			defer wg.Done()
			for result := range channel {
				select {
				case <-ctx.Done():
					return
				default:
					combined <- result
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(combined)
	}()

	return combined
}
