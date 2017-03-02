package commands

import (
	"os/exec"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Status struct{
	BaseCommand
}

func (cmd *Status) Commands() cli.Command {
	return cli.Command{
		Name:   "status",
		Usage:  "Status of the Docker Machine",
		Before: cmd.Before,
		Action: cmd.Run,
	}
}

func (cmd *Status) Run(c *cli.Context) error {
	if !cmd.machine.Exists() {
		cmd.out.Error.Fatalf("No machine named '%s' exists.", cmd.machine.Name)
	}

	util.StreamCommand(exec.Command("docker-machine", "ls", "--filter", "name="+cmd.machine.Name))

	return nil
}
