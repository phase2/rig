package commands

import (
	"fmt"
	"time"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Restart is the command for shutting down and starting a Docker Machine
type Restart struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Restart) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "restart",
			Usage:  "Restart the docker-machine",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig restart` command
func (cmd *Restart) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() || cmd.machine.Exists() {
		if util.SupportsNativeDocker() {
			cmd.out.Spin("Restarting Outrigger services")
		} else {
			cmd.out.Spin(fmt.Sprintf("Restarting Outrigger machine '%s' and services", cmd.machine.Name))
		}

		stop := Stop{cmd.BaseCommand}
		if err := stop.Run(c); err != nil {
			return err
		}

		time.Sleep(time.Duration(5) * time.Second)

		start := Start{cmd.BaseCommand}
		if err := start.Run(c); err != nil {
			return err
		}
	} else {
		return cmd.Failure(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	return cmd.Success("Restart successful")
}
