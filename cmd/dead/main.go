package main

import (
	"os"

	"go.followtheprocess.codes/dead/internal/cmd"
	"go.followtheprocess.codes/msg"
)

func main() {
	if err := run(); err != nil {
		msg.Err(err)
		os.Exit(1)
	}
}

func run() error {
	cli, err := cmd.Build()
	if err != nil {
		return err
	}

	return cli.Execute()
}
