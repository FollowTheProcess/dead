// Package extractor defines an interface for extracting URLs from a variety
// of source formats, as well as concrete implementations of this interface for those
// source formats.
package extractor

// Extractor is a link extractor.
type Extractor interface {
	// Extract parses a body of source text, extracting valid URLs to be checked.
	Extract() ([]string, error)
}
