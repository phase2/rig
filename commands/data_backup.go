package commands

import (
	"fmt"
	"os"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// DataBackup is the command for backing up the /data directory within the Docker Machine
type DataBackup struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *DataBackup) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "data-backup",
			Usage: "Backup the contents of the /data volume of a docker machine",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "data-dir",
					Value: "/mnt/sda1/data",
					Usage: "Specify the directory on the Docker Machine to backup. Defaults to the entire /data volume.",
				},
				cli.StringFlag{
					Name:  "backup-dir",
					Value: fmt.Sprintf("%s%c%s", os.Getenv("HOME"), os.PathSeparator, "rig-backups"),
					Usage: "Specify the local directory to store the backup zip.",
				},
			},
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig data-backup` command
func (cmd *DataBackup) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		return cmd.Success("Data Backup is not needed on Linux, please archive any data directly")
	}

	if !cmd.machine.Exists() {
		return cmd.Failure(fmt.Sprintf("No machine named '%s' exists.", cmd.machine.Name), "MACHINE-NOT-FOUND", 12)
	}

	dataDir := c.String("data-dir")
	backupDir := c.String("backup-dir")
	backupFile := fmt.Sprintf("%s%c%s.tgz", backupDir, os.PathSeparator, cmd.machine.Name)
	if _, err := os.Stat(backupDir); err != nil {
		cmd.out.Info("Creating backup directory: %s...", backupDir)
		if mkdirErr := util.Command("mkdir", "-p", backupDir).Run(); mkdirErr != nil {
			cmd.out.Error(mkdirErr.Error())
			return cmd.Failure(fmt.Sprintf("Could not create backup directory %s", backupDir), "BACKUP-DIR-CREATE-FAILED", 12)
		}
	} else if _, err := os.Stat(backupFile); err == nil {
		// If the backup dir already exists, make sure the backup file does not exist.
		return cmd.Failure(fmt.Sprintf("Backup archive %s already exists.", backupFile), "BACKUP-ARCHIVE-EXISTS", 12)
	}

	// Stream the archive to stdout and capture it in a local file so we don't waste
	// space storing an archive on the VM filesystem. There may not be enough space.
	cmd.out.Spin(fmt.Sprintf("Backing up %s on '%s' to %s...", dataDir, cmd.machine.Name, backupFile))
	archiveCmd := fmt.Sprintf("sudo tar czf - -C %s .", dataDir)
	if err := util.StreamCommand("docker-machine", "ssh", cmd.machine.Name, archiveCmd, ">", backupFile); err != nil {
		cmd.out.Error("Backup failed: %s", err.Error())
		return cmd.Failure("Backup failed", "COMMAND-ERROR", 13)
	}

	cmd.out.Info("Data backup saved to %s", backupFile)
	// Our final success message provides details on where to find the backup file.
	// The success notifcation is kept simple by not passing back the filepath.
	cmd.out.NoSpin()

	return cmd.Success("Data Backup completed")
}
