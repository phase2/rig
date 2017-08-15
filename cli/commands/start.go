package commands

import (
	"os/exec"
	"runtime"
	"strconv"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Start struct {
	BaseCommand
}

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

func (cmd *Start) Run(c *cli.Context) error {
	if runtime.GOOS == "linux" {
		cmd.out.Info.Println("Linux users should use Docker natively for best performance.")
		cmd.out.Info.Println("Please ensure your local Docker setup is compatible with Outrigger.")
		cmd.out.Info.Println("See http://docs.outrigger.sh/getting-started/linux-installation/")
	} else {
		cmd.out.Info.Printf("Starting Docker inside a machine with name '%s'", cmd.machine.Name)
		cmd.out.Verbose.Println("If something goes wrong, run 'rig doctor'")
		cmd.out.Verbose.Println("Pre-flight check...")

		if err := exec.Command("grep", "-qE", "'^\"?/Users/'", "/etc/exports").Run(); err == nil {
			cmd.out.Error.Fatal("Vagrant NFS mount found. Please remove any non-Outrigger mounts that begin with /Users from your /etc/exports file")
		}

		cmd.out.Verbose.Println("Resetting Docker environment variables...")
		cmd.machine.UnsetEnv()

		// Does the docker-machine exist
		if !cmd.machine.Exists() {
			cmd.out.Warning.Printf("No machine named '%s' exists", cmd.machine.Name)

			driver := c.String("driver")
			diskSize := strconv.Itoa(c.Int("disk-size") * 1000)
			memSize := strconv.Itoa(c.Int("memory-size"))
			cpuCount := strconv.Itoa(c.Int("cpu-count"))
			cmd.machine.Create(driver, cpuCount, memSize, diskSize)
		}

		cmd.machine.Start()

		cmd.out.Verbose.Println("Configuring the local Docker environment")
		cmd.machine.SetEnv()

		cmd.out.Info.Println("Setting up DNS...")
		dns := Dns{BaseCommand{machine: cmd.machine, out: cmd.out}}
		dns.ConfigureDns(cmd.machine, c.String("nameservers"))

		// NFS mounts are Mac-only.
		if runtime.GOOS == "darwin" {
			cmd.out.Verbose.Println("Enabling NFS file sharing")
			if nfsErr := util.StreamCommand(exec.Command("docker-machine-nfs", cmd.machine.Name)); nfsErr != nil {
				cmd.out.Error.Printf("Error enabling NFS: %s", nfsErr)
			}
			cmd.out.Verbose.Println("NFS is ready to use")
		}

		// NFS enabling may have caused a machine restart, wait for it to be available before proceeding
		cmd.machine.WaitForDev()

		cmd.out.Verbose.Println("Setting up persistent /data volume...")
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
		util.StreamCommand(exec.Command("docker-machine", "ssh", cmd.machine.Name, dataMountSetup))

		dns.ConfigureRoutes(cmd.machine)

		cmd.out.Verbose.Println("Use docker-machine to interact with your virtual machine.")
		cmd.out.Verbose.Printf("For example, to SSH into it: docker-machine ssh %s", cmd.machine.Name)
		cmd.out.Info.Println("To run Docker commands, your terminal session should be initialized with: 'eval \"$(rig config)\"'")
	}

	cmd.out.Info.Println("Launching Dashboard...")
	dash := Dashboard{BaseCommand{machine: cmd.machine, out: cmd.out}}
	dash.LaunchDashboard(cmd.machine)

	cmd.out.Info.Println("Outrigger is ready to use")

	return nil
}
