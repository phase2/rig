package commands

import (
	"time"

	"github.com/urfave/cli"
)

// Dev is the command for setting docker config to talk to a Docker Machine
type Dev struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Dev) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "dev:win",
			Usage:  "A no-op command that will always succeed.",
			Before: cmd.Before,
			Action: cmd.RunSucceed,
			Hidden: true,
		},
		{
			Name:   "dev:fail",
			Usage:  "A no-op command that will always fail.",
			Before: cmd.Before,
			Action: cmd.RunFail,
			Hidden: true,
		},
	}
}

// RunSucceed executes the `rig dev:succeed` command
func (cmd *Dev) RunSucceed(c *cli.Context) error {
	cmd.out.Spin("Think positive...")
	time.Sleep(3 * time.Second)
	cmd.out.Info("We've got it.")
	return cmd.Success("Positively successful!")
}

// RunFail executes the `rig dev:fail` command
func (cmd *Dev) RunFail(c *cli.Context) error {
	cmd.out.Spin("Abandon all hope...")
	time.Sleep(3 * time.Second)
	cmd.out.Warning("Hope slipping...")
	cmd.out.Spin("Is the sky painted black?")
	time.Sleep(3 * time.Second)
	return cmd.Failure("Hope abandoned :(", "ABANDON-HOPE", 418)
}
