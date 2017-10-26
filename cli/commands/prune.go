package commands

import (
	"os/exec"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Prune struct {
	BaseCommand
}

func (cmd *Prune) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "prune",
			Usage:  "Cleanup docker dangling images and exited containers",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

func (cmd *Prune) Run(c *cli.Context) error {
	cmd.out.Info.Println("Cleaning up Docker images and containers...")
	if exitCode := util.PassthruCommand(exec.Command("docker", "system", "prune", "--all", "--volumes")); exitCode != 0 {
		return cmd.Error("Error pruning Docker resources.", "COMMAND-ERROR", 13)
	}

	return cmd.Success("")
}
