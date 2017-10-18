package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

type DataBackup struct {
	BaseCommand
}

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

func (cmd *DataBackup) Run(c *cli.Context) error {
	if !cmd.machine.Exists() {
		cmd.out.Error.Fatalf("No machine named '%s' exists.", cmd.machine.Name)
	}

	dataDir := c.String("data-dir")
	backupDir := c.String("backup-dir")
	backupFile := fmt.Sprintf("%s%c%s.tgz", backupDir, os.PathSeparator, cmd.machine.Name)
	if _, err := os.Stat(backupDir); err != nil {
		cmd.out.Info.Printf("Creating backup directory: %s...", backupDir)
		if mkdirErr := exec.Command("mkdir", "-p", backupDir).Run(); mkdirErr != nil {
			cmd.out.Error.Println(mkdirErr)
			return cmd.Error(fmt.Sprintf("Could not create backup directory %s", backupDir), 12)
		}
	} else if _, err := os.Stat(backupFile); err == nil {
		// If the backup dir already exists, make sure the backup file does not exist.
		return cmd.Error(fmt.Sprintf("Backup archive %s already exists.", backupFile), 13)
	}

	cmd.out.Info.Printf("Backing up %s on '%s' to %s...", dataDir, cmd.machine.Name, backupFile)

	// Stream the archive to stdout and capture it in a local file so we don't waste
	// space storing an archive on the VM filesystem. There may not be enough space.
	archiveCmd := fmt.Sprintf("sudo tar czf - -C %s .", dataDir)
	backup := exec.Command("docker-machine", "ssh", cmd.machine.Name, archiveCmd, ">", backupFile)
	backup.Stderr = os.Stderr

	color.Set(color.FgCyan)
	err := backup.Run()
	color.Unset()

	if err != nil {
		return cmd.Error(err.Error(), 14)
	}

	return cmd.Success("Data Backup completed with no errors")
}
