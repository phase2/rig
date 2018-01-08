package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Config is the command for setting docker config to talk to a Docker Machine
type Config struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Config) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "config",
			Usage:  "Echo the config to setup the Rig environment.  Run: eval \"$(rig config)\"",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig config` command
func (cmd *Config) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.Success("Config is not needed on Linux")
	}

	// Darwin is installed via brew, so no need to muck with PATH
	if !util.IsMac() {
		// Add stuff to PATH only once
		path := os.Getenv("PATH")
		dir, _ := util.GetExecutableDir()
		if !strings.Contains(path, dir) {
			fmt.Printf("export PATH=%s%c$PATH\n", dir, os.PathListSeparator)
		}
	}

	// Clear out any previous environment variables
	if output, err := util.Command("docker-machine", "env", "-u").Output(); err == nil {
		os.Stdout.Write(output)
	}

	if cmd.machine.Exists() {
		// Setup new values if machine is running
		if output, err := util.Command("docker-machine", "env", cmd.machine.Name).Output(); err == nil {
			os.Stdout.Write(output)
		}
	} else {
		return cmd.Failure(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	return cmd.Success("")
}
