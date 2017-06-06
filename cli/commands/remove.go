package commands

import (
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Remove struct {
	BaseCommand
}

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

func (cmd *Remove) Run(c *cli.Context) error {
	if !cmd.machine.Exists() {
		cmd.out.Error.Fatalf("No machine named '%s' exists.", cmd.machine.Name)
	}

	cmd.out.Info.Printf("Removing '%s'", cmd.machine.Name)

	force := c.Bool("force")
	if !force {
		cmd.out.Warning.Println("!!!!! This operation is destructive. You may lose important data. !!!!!!!")
		cmd.out.Warning.Println("Run 'rig data-backup' if you want to save your /data volume.")
		cmd.out.Warning.Println()

		if !util.AskYesNo("Are you sure you want to remove '" + cmd.machine.Name + "'") {
			cmd.out.Warning.Println("Remove was aborted")
			return nil
		}
	}

	// Run kill first
	kill := Kill{BaseCommand{machine: cmd.machine, out: cmd.out}}
	kill.Run(c)

	cmd.out.Info.Println("Removing the docker-machine")
	cmd.machine.Remove()

	return nil
}
