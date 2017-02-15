package main

import (
	"os/exec"

	"github.com/urfave/cli"
)

type Prune struct{}

func (cmd *Prune) Commands() cli.Command {
	return cli.Command{
		Name:   "prune",
		Usage:  "Cleanup docker dangling images and exited containers",
		Action: cmd.Run,
	}
}

func (cmd *Prune) Run(c *cli.Context) error {
	out.Info.Println("Removing exited Docker containers...")
	StreamCommand(exec.Command("bash", "-c", "docker ps -a -q -f status=exited | grep . | xargs docker rm -v"))

	out.Info.Println("Removing dangling Docker images...")
	StreamCommand(exec.Command("bash", "-c", "docker images --no-trunc -q -f \"dangling=true\" | grep . | xargs docker rmi"))

	return nil
}
