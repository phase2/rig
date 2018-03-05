package commands

import (
	"fmt"
	"os/exec"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Start is the command for creating and starting a Docker Machine and other core Outrigger services
type Ssh struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Ssh) Commands() []cli.Command {
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
func (cmd *Ssh) Run(c *cli.Context) error {
	// Does the docker-machine exist
	if !cmd.machine.Exists() {
		return fmt.Errorf("docker machine %s not found", cmd.machine.Name)
	}

	exitCode := util.PassthruCommand(exec.Command("docker-machine", "ssh", cmd.machine.Name))
	if exitCode == 0 {
		return nil
	} else {
		return fmt.Errorf("Exit code was %d", exitCode)
	}
}
