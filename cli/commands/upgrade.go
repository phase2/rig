package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Upgrade struct{
	BaseCommand
}

func (cmd *Upgrade) Commands() cli.Command {
	return cli.Command{
		Name:  "upgrade",
		Usage: "Upgrade the Docker Machine to a newer/compatible version",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "data-dir",
				Value: "/mnt/sda1/data",
				Usage: "Specify the directory on the Docker Machine to backup. Defaults to the entire /data volume.",
			},
			cli.StringFlag{
				Name:  "backup-dir",
				Value: fmt.Sprintf("%s%c%s%c%s", os.Getenv("HOME"), os.PathSeparator, "rig-backups", os.PathSeparator, "upgrade"),
				Usage: "Specify the local directory to store the backup zip.",
			},
		},
		Before: cmd.Before,
		Action: cmd.Run,
	}
}

func (cmd *Upgrade) Run(c *cli.Context) error {
	cmd.out.Info.Printf("Upgrading '%s'...", cmd.machine.Name)

	if cmd.machine.GetData().Get("Driver").Get("Boot2DockerURL").MustString() == "" {
		cmd.out.Info.Printf("Machine '%s' was not created with a boot2docker URL. Run `docker-machine upgrade %s` directly", cmd.machine.Name, cmd.machine.Name)
		os.Exit(1)
	}

	currentDockerVersion := util.GetCurrentDockerVersion()
	machineDockerVersion, err := cmd.machine.GetDockerVersion()
	if err != nil {
		cmd.out.Error.Fatalf("Could not determine Machine Docker version. Is your machine running?. %s", err)
	}

	if currentDockerVersion.Equal(machineDockerVersion) {
		cmd.out.Info.Printf("Machine '%s' has the same Docker version (%s) as your local Docker binary (%s). There is nothing to upgrade. If you wish to upgrade you'll need to install a newer version of the Docker binary before running the upgrade command.", cmd.machine.Name, machineDockerVersion, currentDockerVersion)
		os.Exit(1)
	}

	cmd.out.Info.Printf("Backing up to prepare for upgrade...")
	backup := &DataBackup{BaseCommand{machine: cmd.machine, out: cmd.out}}
	backup.Run(c)

	remove := &Remove{BaseCommand{machine: cmd.machine, out: cmd.out}}
	removeCtx := cmd.NewContext(remove.Commands().Name, remove.Commands().Flags, c)
	cmd.SetContextFlag(removeCtx, "force", strconv.FormatBool(true))
	remove.Run(removeCtx)

	start := &Start{BaseCommand{machine: cmd.machine, out: cmd.out}}
	startCtx := cmd.NewContext(start.Commands().Name, start.Commands().Flags, c)
	cmd.SetContextFlag(startCtx, "driver", cmd.machine.GetDriver())
	cmd.SetContextFlag(startCtx, "cpu-count", strconv.FormatInt(int64(cmd.machine.GetCPU()), 10))
	cmd.SetContextFlag(startCtx, "memory-size", strconv.FormatInt(int64(cmd.machine.GetMemory()), 10))
	cmd.SetContextFlag(startCtx, "disk-size", strconv.FormatInt(int64(cmd.machine.GetDiskInGB()), 10))
	start.Run(startCtx)

	restore := &DataRestore{BaseCommand{machine: cmd.machine, out: cmd.out}}
	restoreCtx := cmd.NewContext(restore.Commands().Name, restore.Commands().Flags, c)
	cmd.SetContextFlag(restoreCtx, "data-dir", c.String("data-dir"))
	backupFile := fmt.Sprintf("%s%c%s.tgz", c.String("backup-dir"), os.PathSeparator, cmd.machine.Name)
	cmd.SetContextFlag(restoreCtx, "backup-file", backupFile)
	restore.Run(restoreCtx)

	return nil
}
