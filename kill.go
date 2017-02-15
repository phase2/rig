package main

import (
	"os/exec"

	"github.com/urfave/cli"
)

type Kill struct{}

func (cmd *Kill) Commands() cli.Command {
	return cli.Command{
		Name:   "kill",
		Usage:  "Kill the docker-machine",
		Action: cmd.Run,
	}
}

func (cmd *Kill) Run(c *cli.Context) error {
	if !machine.Exists() {
		out.Error.Fatalf("No machine named '%s' exists.", machine.Name)
	}

	// First stop it (and cleanup)
	stop := Stop{}
	stop.Run(c)

	out.Info.Printf("Killing machine '%s'", machine.Name)
	StreamCommand(exec.Command("docker-machine", "kill", machine.Name))

	// Ensure the underlying virtualization has stopped
	driver := machine.GetDriver()
	switch driver {
	case "virtualbox":
		StreamCommand(exec.Command("controlvm", machine.Name, "poweroff"))
	case "vmwarefusion":
		out.Warning.Println("Add vmrun suspend command.")
	case "xhyve":
		out.Warning.Println("Add equivalent xhyve kill command.")
	default:
		out.Warning.Printf("Driver not recognized: %s\n", driver)
	}

	return nil
}
