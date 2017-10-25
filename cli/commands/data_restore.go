package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

type DataRestore struct {
	BaseCommand
}

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

func (cmd *DataRestore) Run(c *cli.Context) error {
	if !cmd.machine.Exists() {
		cmd.out.Error.Fatalf("No machine named '%s' exists.", cmd.machine.Name)
	}

	dataDir := c.String("data-dir")
	backupFile := strings.TrimSpace(c.String("backup-file"))
	if len(backupFile) == 0 {
		backupFile = fmt.Sprintf("%s%c%s%c%s.tgz", os.Getenv("HOME"), os.PathSeparator, "rig-backups", os.PathSeparator, cmd.machine.Name)
	}

	if _, err := os.Stat(backupFile); err != nil {
		return cmd.Error(fmt.Sprintf("Backup archive %s doesn't exists.", backupFile), "BACKUP-ARCHIVE-NOT-FOUND", 12)
	}

	cmd.out.Info.Printf("Restoring %s to %s on '%s'...", backupFile, dataDir, cmd.machine.Name)

	// Send the archive via stdin and extract inline. Saves on disk & performance
	extractCmd := fmt.Sprintf("cat %s | docker-machine ssh %s \"sudo tar xzf - -C %s\"", backupFile, cmd.machine.Name, dataDir)
	cmd.out.Info.Printf(extractCmd)
	backup := exec.Command("bash", "-c", extractCmd)
	backup.Stderr = os.Stderr

	color.Set(color.FgCyan)
	err := backup.Run()
	color.Unset()

	if err != nil {
		return cmd.Error(err.Error(), "COMMAND-ERROR", 13)
	}

	return cmd.Success("Data Restore was successful")
}
