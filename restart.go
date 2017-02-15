package main

import (
	"time"

	"github.com/urfave/cli"
)

type Restart struct{}

func (cmd *Restart) Commands() cli.Command {
	return cli.Command{
		Name:   "restart",
		Usage:  "Restart the docker-machine",
		Action: cmd.Run,
	}
}

func (cmd *Restart) Run(c *cli.Context) error {
	if machine.Exists() {
		out.Info.Printf("Restarting machine '%s'", machine.Name)

		stop := Stop{}
		stop.Run(c)

		time.Sleep(time.Duration(5) * time.Second)

		start := Start{}
		start.Run(c)
	} else {
		out.Error.Fatalf("No machine named '%s' exists.", machine.Name)
	}

	return nil
}
