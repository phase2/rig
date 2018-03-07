package commands

import (
	"fmt"
	"os/exec"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// SSH is the command for staring an SSH session inside the docker machine vm
type SSH struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *SSH) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "ssh",
			Usage:  "Start an ssh session into the docker-machine vm",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig ssh` command
func (cmd *SSH) Run(c *cli.Context) error {
	// Does the docker-machine exist
	if !cmd.machine.Exists() {
		return cmd.Failure(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	/* #nosec */
	if exitCode := util.PassthruCommand(exec.Command("docker-machine", "ssh", cmd.machine.Name)); exitCode != 0 {
		return cmd.Failure(fmt.Sprintf("Failure running 'docker-machine ssh %s'", cmd.machine.Name), "COMMAND-ERROR", exitCode)
	}
	return cmd.Success("")
}
