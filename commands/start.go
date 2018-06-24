package commands

import (
	"fmt"
	"strconv"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Start is the command for creating and starting a Docker Machine and other core Outrigger services
type Start struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Start) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:  "start",
			Usage: "Start the docker-machine and container services",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "driver",
					Value: "virtualbox",
					Usage: "Which virtualization driver to use: virtualbox (default), vmwarefusion, xhyve. Only used if start needs to create a machine",
				},
				cli.IntFlag{
					Name:  "disk-size",
					Value: 40,
					Usage: "Size of the VM disk in GB. Defaults to 40. Only used if start needs to create a machine.",
				},
				cli.IntFlag{
					Name:  "memory-size",
					Value: 4096,
					Usage: "Amount of memory for the VM in MB. Defaults to 4096. Only used if start needs to create a machine.",
				},
				cli.IntFlag{
					Name:  "cpu-count",
					Value: 2,
					Usage: "Number of CPU to allocate to the VM. Defaults to 2. Only used if start needs to create a machine.",
				},
				cli.StringFlag{
					Name:   "nameservers",
					Value:  "8.8.8.8:53",
					Usage:  "Comma separated list of fallback names servers for DNS resolution.",
					EnvVar: "RIG_NAMESERVERS",
				},
			},
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig start` command
func (cmd *Start) Run(c *cli.Context) error {
	if util.SupportsNativeDocker() {
		cmd.out.Info("Linux users should use Docker natively for best performance.")
		cmd.out.Info("Please ensure your local Docker setup is compatible with Outrigger.")
		cmd.out.Info("See http://docs.outrigger.sh/getting-started/linux-installation/")
		return cmd.StartMinimal(c.String("nameservers"))
	}

	cmd.out.Spin(fmt.Sprintf("Starting Docker & Docker Machine (%s)", cmd.machine.Name))
	cmd.out.Verbose("If something goes wrong, run 'rig doctor'")

	cmd.out.Verbose("Pre-flight check...")

	if err := util.Command("grep", "-qE", "'^\"?/Users/'", "/etc/exports").Run(); err == nil {
		cmd.out.Error("Docker could not be started")
		return cmd.Failure("Vagrant NFS mount found. Please remove any non-Outrigger mounts that begin with /Users from your /etc/exports file", "NFS-MOUNT-CONFLICT", 12)
	}

	cmd.out.Verbose("Resetting Docker environment variables...")
	cmd.machine.UnsetEnv()

	// Does the docker-machine exist
	if !cmd.machine.Exists() {
		cmd.out.Spin(fmt.Sprintf("Creating Docker & Docker Machine (%s)", cmd.machine.Name))
		driver := c.String("driver")
		diskSize := strconv.Itoa(c.Int("disk-size") * 1000)
		memSize := strconv.Itoa(c.Int("memory-size"))
		cpuCount := strconv.Itoa(c.Int("cpu-count"))
		cmd.machine.Create(driver, cpuCount, memSize, diskSize)
	}

	if err := cmd.machine.Start(); err != nil {
		cmd.out.Error("Docker could not be started")
		return cmd.Failure(err.Error(), "MACHINE-START-FAILED", 12)
	}

	cmd.out.Verbose("Configuring the local Docker environment")
	cmd.machine.SetEnv()
	cmd.out.Info("Docker Machine (%s) Created", cmd.machine.Name)

	dns := DNS{cmd.BaseCommand}
	dns.StartDNS(cmd.machine, c.String("nameservers"))

	// NFS mounts are Mac-only.
	if util.IsMac() {
		cmd.out.Spin("Enabling NFS file sharing...")
		if nfsErr := util.StreamCommand("docker-machine-nfs", cmd.machine.Name); nfsErr != nil {
			cmd.out.Warning("Failure enabling NFS: %s", nfsErr.Error())
		} else {
			cmd.out.Info("NFS is ready")
		}
	}

	cmd.out.Spin("Preparing /data filesystem...")
	// NFS enabling may have caused a machine restart, wait for it to be available before proceeding
	if err := cmd.machine.WaitForDev(); err != nil {
		return cmd.Failure(err.Error(), "MACHINE-START-FAILED", 12)
	}

	cmd.out.Verbose("Setting up persistent /data volume...")
	dataMountSetup := `if [ ! -d /mnt/sda1/data ];
		then echo '===> Creating /mnt/sda1/data directory';
		sudo mkdir /mnt/sda1/data;
		sudo chgrp staff /mnt/sda1/data;
		sudo chmod g+w /mnt/sda1/data;
		echo '===> Creating /var/lib/boot2docker/bootsync.sh';
		echo '#!/bin/sh' | sudo tee /var/lib/boot2docker/bootsync.sh > /dev/null;
		echo 'sudo ln -sf /mnt/sda1/data /data' | sudo tee -a /var/lib/boot2docker/bootsync.sh > /dev/null;
		sudo chmod +x /var/lib/boot2docker/bootsync.sh;
	fi;
	if [ ! -L /data ];
		then echo '===> Creating symlink from /data to /mnt/sda1/data';
		sudo ln -s /mnt/sda1/data /data;
	fi;`
	if err := util.StreamCommand("docker-machine", "ssh", cmd.machine.Name, dataMountSetup); err != nil {
		return cmd.Failure(err.Error(), "DATA-MOUNT-FAILED", 13)
	}
	cmd.out.Info("/data filesystem is ready")

	// Route configuration needs to be finalized after NFS-triggered reboots.
	// This rebooting may change key details such as IP Address of the Dev machine.
	dns.ConfigureRoutes(cmd.machine)

	cmd.out.Verbose("Use docker-machine to interact with your virtual machine.")
	cmd.out.Verbose("For example, to SSH into it: docker-machine ssh %s", cmd.machine.Name)

	cmd.out.Spin("Launching Dashboard...")
	dash := Dashboard{cmd.BaseCommand}
	dash.LaunchDashboard(cmd.machine)
	cmd.out.Info("Dashboard is ready")

	cmd.out.Info("Run 'eval \"$(rig config)\"' to execute docker or docker-compose commands in your terminal.")
	return cmd.Success("Outrigger is ready to use")
}

// StartMinimal will start "minimal" Outrigger operations, which refers to environments where
// a virtual machine and networking is not required or managed by Outrigger.
func (cmd *Start) StartMinimal(nameservers string) error {
	dns := DNS{cmd.BaseCommand}
	dns.StartDNS(cmd.machine, nameservers)

	dash := Dashboard{cmd.BaseCommand}
	dash.LaunchDashboard(cmd.machine)

	return cmd.Success("Outrigger services started")
}
