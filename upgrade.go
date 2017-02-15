package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/urfave/cli"
)

type Upgrade struct{}

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
		Action: cmd.Run,
	}
}

func (cmd *Upgrade) Run(c *cli.Context) error {
	out.Info.Printf("Upgrading '%s'...", machine.Name)

	if machine.GetData().Get("Driver").Get("Boot2DockerURL").MustString() == "" {
		out.Info.Printf("Machine '%s' was not created with a boot2docker URL. Run `docker-machine upgrade %s` directly", machine.Name, machine.Name)
		os.Exit(1)
	}

	currentDockerVersion := GetCurrentDockerVersion()
	machineDockerVersion, err := machine.GetDockerVersion()
	if err != nil {
		out.Error.Fatalf("Could not determine Machine Docker version. Is your machine running?. %s", err)
	}

	if currentDockerVersion.Equal(machineDockerVersion) {
		out.Info.Printf("Machine '%s' has the same Docker version (%s) as your local Docker binary (%s). There is nothing to upgrade. If you wish to upgrade you'll need to install a newer version of the Docker binary before running the upgrade command.", machine.Name, machineDockerVersion, currentDockerVersion)
		os.Exit(1)
	}

	out.Info.Printf("Backing up to prepare for upgrade...")
	backup := &DataBackup{}
	backup.Run(c)

	remove := &Remove{}
	removeCtx := NewContext(remove, c)
	SetContextFlag(removeCtx, "force", strconv.FormatBool(true))
	remove.Run(removeCtx)

	start := &Start{}
	startCtx := NewContext(start, c)
	SetContextFlag(startCtx, "driver", machine.GetDriver())
	SetContextFlag(startCtx, "cpu-count", strconv.FormatInt(int64(machine.GetCPU()), 10))
	SetContextFlag(startCtx, "memory-size", strconv.FormatInt(int64(machine.GetMemory()), 10))
	SetContextFlag(startCtx, "disk-size", strconv.FormatInt(int64(machine.GetDiskInGB()), 10))
	start.Run(startCtx)

	restore := &DataRestore{}
	restoreCtx := NewContext(restore, c)
	SetContextFlag(restoreCtx, "data-dir", c.String("data-dir"))
	backupFile := fmt.Sprintf("%s%c%s.tgz", c.String("backup-dir"), os.PathSeparator, machine.Name)
	SetContextFlag(restoreCtx, "backup-file", backupFile)
	restore.Run(restoreCtx)

	return nil
}
