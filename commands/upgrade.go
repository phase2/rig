package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Upgrade is the command for backing up Docker Machine /data, upgrading the Docker Machine
// to the most recent release and restoring the Docker Machine /data
type Upgrade struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Upgrade) Commands() []cli.Command {
	return []cli.Command{
		{
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
		},
	}
}

// Run executes the `rig upgrade` command
func (cmd *Upgrade) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.Success("Upgrade is not needed on Linux")
	}

	cmd.out.Spin(fmt.Sprintf("Upgrading '%s'...", cmd.machine.Name))

	if cmd.machine.GetData().Get("Driver").Get("Boot2DockerURL").MustString() == "" {
		cmd.out.Error("Machine %s not compatible with rig upgrade", cmd.machine.Name)
		return cmd.Failure(fmt.Sprintf("Machine '%s' was not created with a boot2docker URL. Run `docker-machine upgrade %s` directly", cmd.machine.Name, cmd.machine.Name), "MACHINE-CREATED-MANUALLY", 12)
	}

	currentDockerVersion := util.GetCurrentDockerVersion()
	machineDockerVersion, err := cmd.machine.GetDockerVersion()
	if err != nil {
		return cmd.Failure(fmt.Sprintf("Could not determine Machine Docker version. Is your machine running?. %s", err), "MACHINE-STOPPED", 12)
	}

	if currentDockerVersion.Equal(machineDockerVersion) {
		return cmd.Success(fmt.Sprintf("Machine '%s' has the same Docker version (%s) as your local Docker binary (%s). There is nothing to upgrade. If you wish to upgrade you'll need to install a newer version of the Docker binary before running the upgrade command.", cmd.machine.Name, machineDockerVersion, currentDockerVersion))
	}

	cmd.out.Info("Backing up to prepare for upgrade...")
	backup := &DataBackup{cmd.BaseCommand}
	if err := backup.Run(c); err != nil {
		return err
	}

	remove := &Remove{cmd.BaseCommand}
	removeCtx := cmd.NewContext(remove.Commands()[0].Name, remove.Commands()[0].Flags, c)
	cmd.SetContextFlag(removeCtx, "force", strconv.FormatBool(true))
	if err := remove.Run(removeCtx); err != nil {
		return err
	}

	start := &Start{cmd.BaseCommand}
	startCtx := cmd.NewContext(start.Commands()[0].Name, start.Commands()[0].Flags, c)
	cmd.SetContextFlag(startCtx, "driver", cmd.machine.GetDriver())
	cmd.SetContextFlag(startCtx, "cpu-count", strconv.FormatInt(int64(cmd.machine.GetCPU()), 10))
	cmd.SetContextFlag(startCtx, "memory-size", strconv.FormatInt(int64(cmd.machine.GetMemory()), 10))
	cmd.SetContextFlag(startCtx, "disk-size", strconv.FormatInt(int64(cmd.machine.GetDiskInGB()), 10))
	if err := start.Run(startCtx); err != nil {
		return err
	}

	restore := &DataRestore{cmd.BaseCommand}
	restoreCtx := cmd.NewContext(restore.Commands()[0].Name, restore.Commands()[0].Flags, c)
	cmd.SetContextFlag(restoreCtx, "data-dir", c.String("data-dir"))
	backupFile := fmt.Sprintf("%s%c%s.tgz", c.String("backup-dir"), os.PathSeparator, cmd.machine.Name)
	cmd.SetContextFlag(restoreCtx, "backup-file", backupFile)
	if err := restore.Run(restoreCtx); err != nil {
		return err
	}

	return cmd.Success(fmt.Sprintf("Upgrade of '%s' complete", cmd.machine.Name))
}
