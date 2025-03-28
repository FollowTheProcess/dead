package main

import (
	"fmt"
	"os"
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
	})
}
