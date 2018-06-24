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

	if util.AskYesNo("Are you sure you want to remove all unused containers, networks, images, caches, and volumes?") {
		cmd.out.Info("Cleaning up unused Docker resources. This may take a while...")
		/* #nosec */
		if exitCode := util.PassthruCommand(exec.Command("docker", "system", "prune", "--all", "--volumes", "--force")); exitCode != 0 {
			return cmd.Failure("Failure pruning Docker resources.", "COMMAND-ERROR", 13)
		}
		cmd.out.Info("Unused Docker images, containers, volumes, and networks cleaned up.")
	} else {
		cmd.out.Warn("Cleanup aborted.")
	}

	return cmd.Success("")
}
