package main

import (
	"github.com/urfave/cli"
)

type Remove struct{}

func (cmd *Remove) Commands() cli.Command {
	return cli.Command{
		Name:  "remove",
		Usage: "Remove the docker-machine",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "force, f",
				Usage: "Force removal. Don't prompt to backup /data",
			},
		},
		Action: cmd.Run,
	}
}

func (cmd *Remove) Run(c *cli.Context) error {
	if !machine.Exists() {
		out.Error.Fatalf("No machine named '%s' exists.", machine.Name)
	}

	out.Info.Printf("Removing '%s'", machine.Name)

	force := c.Bool("force")
	if !force {
		out.Warning.Println("!!!!! This operation is destructive. You may lose important data. !!!!!!!")
		out.Warning.Println("Run 'rig data-backup' if you want to save your /data volume.")
		out.Warning.Println()

		if !AskYesNo("Are you sure you want to remove '" + machine.Name + "'") {
			out.Warning.Println("Remove was aborted")
			return nil
		}
	}

	// Run kill first
	kill := Kill{}
	kill.Run(c)

	out.Info.Println("Removing the docker-machine")
	machine.Remove()

	return nil
}
