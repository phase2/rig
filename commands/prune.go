package commands

import (
	"os/exec"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Prune is the command for cleaning up Docker resources
type Prune struct {
	BaseCommand
}

// Commands returns the operations supported by this command
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

// Run executes the `rig prune` command
func (cmd *Prune) Run(c *cli.Context) error {
	cmd.out.Info("Cleaning up Docker images and containers...")
	if exitCode := util.PassthruCommand(exec.Command("docker", "system", "prune", "--all", "--volumes")); exitCode != 0 {
		return cmd.Failure("Failure pruning Docker resources.", "COMMAND-ERROR", 13)
	}

	return cmd.Success("")
}
