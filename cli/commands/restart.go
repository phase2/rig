package commands

import (
	"runtime"
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
	if runtime.GOOS == "linux" || cmd.machine.Exists() {
		if runtime.GOOS == "linux" {
			cmd.out.Info.Println("Restarting Outrigger")
		} else {
			cmd.out.Info.Printf("Restarting Outrigger on Machine '%s'", cmd.machine.Name)
		}

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
		return cmd.Error(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	return cmd.Success(fmt.Sprintf("Machine '%s' restarted", cmd.machine.Name))
}
