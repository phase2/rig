package commands

import (
	"fmt"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Remove is the command for deleting a Docker Machine
type Remove struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Remove) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "remove",
			Usage: "Remove the docker-machine",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force, f",
					Usage: "Force removal. Don't prompt to backup /data",
				},
			},
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig remove` command
func (cmd *Remove) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.Success("Remove is not needed on Linux")
	}

	if !cmd.machine.Exists() {
		return cmd.Error(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	cmd.out.Info.Printf("Removing '%s'", cmd.machine.Name)

	force := c.Bool("force")
	if !force {
		cmd.out.Warning.Println("!!!!! This operation is destructive. You may lose important data. !!!!!!!")
		cmd.out.Warning.Println("Run 'rig data-backup' if you want to save your /data volume.")
		cmd.out.Warning.Println()

		if !util.AskYesNo("Are you sure you want to remove '" + cmd.machine.Name + "'") {
			return cmd.Success("Remove was aborted")
		}
	}

	// Run kill first
	kill := Kill{BaseCommand{machine: cmd.machine, out: cmd.out}}
	if err := kill.Run(c); err != nil {
		return err
	}

	cmd.out.Info.Println("Removing the docker-machine")
	if err := cmd.machine.Remove(); err != nil {
		return cmd.Error(err.Error(), "MACHINE-REMOVE-FAILED", 12)
	}

	return cmd.Success(fmt.Sprintf("Machine '%s' removed", cmd.machine.Name))
}
