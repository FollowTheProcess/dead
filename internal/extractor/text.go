package extractor

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

const urlRegexRaw = `https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*)`

var urlRegex = regexp.MustCompile(urlRegexRaw)

// TextExtractor is an [Extractor] that pulls URLs from arbitrary text using a regex, it
// is probably best to try and use a more specific extractor that does smarter parsing but
// this one should handle any arbitrary text based format.
type TextExtractor struct {
	src io.Reader
}

// NewText returns a new [TextExtractor] reading from r.
func NewText(r io.Reader) TextExtractor {
	return TextExtractor{src: r}
}

// Extract pulls URLs from arbitrary text.
func (t TextExtractor) Extract() ([]string, error) {
	contents, err := io.ReadAll(t.src)
	if err != nil {
		return nil, fmt.Errorf("could not read from input: %w", err)
	}

	matches := urlRegex.FindAllString(string(contents), -1)

	// The regex above captures urls that end in a trailing dot and the regex way of avoiding that
	// involves backtracking so it's most likely quicker to just loop through the matches and remove any trailing
	// dots
	for i := range matches {
		matches[i] = strings.TrimSuffix(matches[i], ".")
	}

	return matches, nil
}
