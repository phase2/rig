package commands

import (
	"time"

	"fmt"
	"github.com/urfave/cli"
)

type Restart struct {
	BaseCommand
}

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

func (cmd *Restart) Run(c *cli.Context) error {
	if cmd.machine.Exists() {
		cmd.out.Info.Printf("Restarting machine '%s'", cmd.machine.Name)

		stop := Stop{BaseCommand{machine: cmd.machine, out: cmd.out}}
		if err := stop.Run(c); err != nil {
			return err
		}

		time.Sleep(time.Duration(5) * time.Second)

		start := Start{BaseCommand{machine: cmd.machine, out: cmd.out}}
		if err := start.Run(c); err != nil {
			return err
		}
	} else {
		return cmd.Error(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), 11)
	}

	return cmd.Success(fmt.Sprintf("Machine '%s' restarted", cmd.machine.Name))
}
