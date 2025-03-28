package extractor_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FollowTheProcess/dead/internal/extractor"
	"github.com/FollowTheProcess/test"
	"github.com/rogpeppe/go-internal/txtar"
)

var ci = os.Getenv("CI") != ""

func TestTextExtractor(t *testing.T) {
	test.ColorEnabled(!ci) // Force colour in diffs, but locally only

	pattern := filepath.Join("testdata", "Text", "*.txtar")
	files, err := filepath.Glob(pattern)
	test.Ok(t, err)

	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			archive, err := txtar.ParseFile(file)
			test.Ok(t, err)

			if len(archive.Files) != 2 {
				t.Fatalf("expected 2 files in the archive, got %d", len(archive.Files))
			}

			srcFile := archive.Files[0]
			expectedFile := archive.Files[1]

			if srcFile.Name != "src" {
				t.Fatalf("expected first file to be named 'src', got %q", srcFile.Name)
			}

			if expectedFile.Name != "expected" {
				t.Fatalf("expected second file to be named 'expected', got %q", expectedFile.Name)
			}

			src := bytes.NewReader(srcFile.Data)
			ext := extractor.NewText(src)

			links, err := ext.Extract()
			test.Ok(t, err)

			got := strings.Join(links, "\n") + "\n" // Add a last newline on the end
			want := string(expectedFile.Data)

			test.Diff(t, got, want)
		})
	}
}
