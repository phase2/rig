package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Status struct {
	BaseCommand
}

func (cmd *Status) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "status",
			Usage:  "Status of the Docker Machine",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

func (cmd *Status) Run(c *cli.Context) error {
	if runtime.GOOS == "linux" {
		return cmd.Success("Status is not needed on Linux")
	}

	if !cmd.machine.Exists() {
		return cmd.Error(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	if cmd.out.IsVerbose {
		util.StreamCommand(exec.Command("docker-machine", "ls", "--filter", "name="+cmd.machine.Name))
	} else {
		output, _ := exec.Command("docker-machine", "status", cmd.machine.Name).CombinedOutput()
		os.Stdout.Write(output)
	}

	return cmd.Success("")
}
