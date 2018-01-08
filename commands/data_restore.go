package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// DataRestore is the command for restoring up the /data directory within the Docker Machine
type DataRestore struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *DataRestore) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "data-restore",
			Usage: "Restore a local backup to the /data volume of a docker machine",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "backup-file",
					Usage: "Specify the local archive to restore to the VM. Defaults to a file named $HOME/rig-backups/<machinename>.tgz",
				},
				cli.StringFlag{
					Name:  "data-dir",
					Value: "/mnt/sda1/data",
					Usage: "Specify the restore dir on the VM. Defaults to the entire /data volume.",
				},
			},
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig data-restore` command
func (cmd *DataRestore) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.Success("Data Restore is not needed on Linux, please unarchive any data directly")
	}

	if !cmd.machine.Exists() {
		return cmd.Failure(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	dataDir := c.String("data-dir")
	backupFile := strings.TrimSpace(c.String("backup-file"))
	if len(backupFile) == 0 {
		backupFile = fmt.Sprintf("%s%c%s%c%s.tgz", os.Getenv("HOME"), os.PathSeparator, "rig-backups", os.PathSeparator, cmd.machine.Name)
	}

	if _, err := os.Stat(backupFile); err != nil {
		return cmd.Failure(fmt.Sprintf("Backup archive %s doesn't exists.", backupFile), "BACKUP-ARCHIVE-NOT-FOUND", 12)
	}

	cmd.out.Spin(fmt.Sprintf("Restoring %s to %s on '%s'...", backupFile, dataDir, cmd.machine.Name))
	// Send the archive via stdin and extract inline. Saves on disk & performance
	extractCmd := fmt.Sprintf("cat %s | docker-machine ssh %s \"sudo tar xzf - -C %s\"", backupFile, cmd.machine.Name, dataDir)
	if err := util.StreamCommand("bash", "-c", extractCmd); err != nil {
		cmd.out.Error("Data restore failed: %s", err.Error())
		return cmd.Failure("Data restore failed", "COMMAND-ERROR", 13)
	}

	return cmd.Success("Data Restore completed")
}
