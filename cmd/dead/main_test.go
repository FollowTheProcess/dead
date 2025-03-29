package main

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"dead": func() {
			if err := run(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	})
}

func Test(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:                 "testdata",
		UpdateScripts:       false,
		RequireExplicitExec: true,
		RequireUniqueNames:  true,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"replace": Replace,
		},
	})
}

// Replace is a testscript command that replaces text in a file by way of a regex
// pattern match, useful for replacing non-deterministic output like UUIDs and durations
// with placeholders to facilitate deterministic comparison in tests.
//
// Usage:
//
//	replace <file> <regex> <replacement>
//
// It cannot be negated, regex must be valid, and the file must be present in the
// txtar archive, including "stdout" and "stderr".
func Replace(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! filter")
	}

	if len(args) != 3 {
		ts.Fatalf("Usage: filter <file> <regex> <replacement>")
	}

	file := ts.MkAbs(args[0])
	ts.Logf("filter file: %s", file)

	stdout := ts.ReadFile("stdout")
	regex := args[1]
	replacement := args[2]

	re, err := regexp.Compile(regex)
	ts.Check(err)

	replaced := re.ReplaceAllString(stdout, replacement)

	_, err = ts.Stdout().Write([]byte(replaced))
	ts.Check(err)
}
