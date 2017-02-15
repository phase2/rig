package main

import (
	"os/exec"

	"github.com/urfave/cli"
)

type Status struct{}

func (cmd *Status) Commands() cli.Command {
	return cli.Command{
		Name:   "status",
		Usage:  "Status of the Docker Machine",
		Action: cmd.Run,
	}
}

func (cmd *Status) Run(c *cli.Context) error {
	if !machine.Exists() {
		out.Error.Fatalf("No machine named '%s' exists.", machine.Name)
	}

	StreamCommand(exec.Command("docker-machine", "ls", "--filter", "name="+machine.Name))

	return nil
}
