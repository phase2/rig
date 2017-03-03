package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Watch struct {
	BaseCommand
}

func (cmd *Watch) Commands() cli.Command {
	return cli.Command{
		Name:      "watch",
		Usage:     "Watch a host directory for changes and forward the event into a Docker Machine",
		ArgsUsage: "<path to watch>",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "ignorefile",
				Usage: "File to use for watch ignores. One ignore per line using the regex format for the fswatch command. If not specified it will look for a file named .rig-watch-ignore in the current working directory and all parent dirs",
			},
		},
		Before: cmd.Before,
		Action: cmd.Run,
	}
}

func (cmd *Watch) Run(c *cli.Context) error {
	if len(c.Args()) == 0 {
		cmd.out.Error.Fatal("Path to watch was not specified.")
	}
	path := c.Args()[0]
	ignore := c.String("ignorefile")

	// If not ignorefile was not specified look for a default named one
	if ignore == "" {
		ignore = cmd.FindIgnoreFile()
	}

	if ignore != "" {
		cmd.out.Info.Printf("Found watch ignore file: %s", ignore)
	}

	cmd.out.Info.Printf("Watching: %s Sending events to %s", path, cmd.machine.Name)

	// Prerequisite checks
	if !cmd.machine.IsRunning() {
		cmd.out.Error.Fatalf("Docker Machine '%s' is not running", cmd.machine.Name)
	}
	// Is fswatch installed locally
	if err := exec.Command("which", "fswatch").Run(); err != nil {
		cmd.out.Error.Fatal("fswatch is not installed. Install it with 'brew install fswatch'")
	}
	// Ensure rsync is installed on the machine
	rsyncSetup := `if [ ! -f /usr/local/bin/rsync  ];
    then echo '===> Installing rsync';
    tce-load -wi rsync
  fi;`
	util.StreamCommand(exec.Command("docker-machine", "ssh", cmd.machine.Name, rsyncSetup))

	archDir, _ := util.GetExecutableDir()
	watchScript := fmt.Sprintf("%s%cdocker-machine-watch-rsync.sh", archDir, os.PathSeparator)
	args := []string{"-m", cmd.machine.Name}
	if ignore != "" {
		args = append(args, "-e", ignore)
	}
	args = append(args, path)

	util.StreamCommand(exec.Command(watchScript, args...))

	return nil
}

func (cmd *Watch) FindIgnoreFile() string {

	for current, _ := os.Getwd(); current != "/"; current = filepath.Dir(current) {
		ignoreFile := fmt.Sprintf("%s%c.rig-watch-ignore", current, os.PathSeparator)
		if _, err := os.Stat(ignoreFile); err == nil {
			return ignoreFile
		}
	}

	return ""
}
