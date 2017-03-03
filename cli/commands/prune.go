package commands

import (
	"os/exec"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Prune struct {
	BaseCommand
}

func (cmd *Prune) Commands() cli.Command {
	return cli.Command{
		Name:   "prune",
		Usage:  "Cleanup docker dangling images and exited containers",
		Before: cmd.Before,
		Action: cmd.Run,
	}
}

func (cmd *Prune) Run(c *cli.Context) error {
	cmd.out.Info.Println("Removing exited Docker containers...")
	util.StreamCommand(exec.Command("bash", "-c", "docker ps -a -q -f status=exited | grep . | xargs docker rm -v"))

	cmd.out.Info.Println("Removing dangling Docker images...")
	util.StreamCommand(exec.Command("bash", "-c", "docker images --no-trunc -q -f \"dangling=true\" | grep . | xargs docker rmi"))

	return nil
}
