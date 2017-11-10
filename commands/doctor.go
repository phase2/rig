package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// Doctor is the command for performing diagnostics on the Outrigger environment
type Doctor struct {
	BaseCommand
}

// Commands returns the operations supported by this command
func (cmd *Doctor) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:   "doctor",
			Usage:  "Troubleshoot the Rig environment",
			Before: cmd.Before,
			Action: cmd.Run,
		},
	}
}

// Run executes the `rig doctor` command
// nolint: gocyclo
func (cmd *Doctor) Run(c *cli.Context) error {
	// 0. Ensure all of rig's dependencies are available in the PATH.
	if err := util.Command("docker", "-h").Start(); err == nil {
		cmd.out.Info.Println("Docker is installed.")
	} else {
		cmd.out.Error.Fatal("Docker (docker) is not installed.")
	}
	if !util.SupportsNativeDocker() {
		if err := util.Command("docker-machine", "-h").Start(); err == nil {
			cmd.out.Info.Println("Docker Machine is installed.")
		} else {
			cmd.out.Error.Fatal("Docker Machine (docker-machine) is not installed.")
		}
	}
	if err := util.Command("docker-compose", "-h").Start(); err == nil {
		cmd.out.Info.Println("Docker Compose is installed.")
	} else {
		cmd.out.Warning.Printf("Docker Compose (docker-compose) is not installed.")
	}

	// 1. Ensure the configured docker-machine matches the set environment.
	if !util.SupportsNativeDocker() {
		if cmd.machine.Exists() {
			if _, isset := os.LookupEnv("DOCKER_MACHINE_NAME"); !isset {
				cmd.out.Error.Fatalf("Docker configuration is not set. Please run 'eval \"$(rig config)\"'.")
			} else if cmd.machine.Name != os.Getenv("DOCKER_MACHINE_NAME") {
				cmd.out.Error.Fatalf("Your environment configuration specifies a different machine. Please re-run as 'rig --name=\"%s\" doctor'.", cmd.machine.Name)
			} else {
				cmd.out.Info.Printf("Docker Machine (%s) name matches your environment configuration.", cmd.machine.Name)
			}
			if output, err := util.Command("docker-machine", "url", cmd.machine.Name).Output(); err == nil {
				hostURL := strings.TrimSpace(string(output))
				if hostURL != os.Getenv("DOCKER_HOST") {
					cmd.out.Error.Fatalf("Docker Host configuration should be '%s' but got '%s'. Please re-run 'eval \"$(rig config)\"'.", os.Getenv("DOCKER_HOST"), hostURL)
				} else {
					cmd.out.Info.Printf("Docker Machine (%s) URL (%s) matches your environment configuration.", cmd.machine.Name, hostURL)
				}
			}
		} else {
			cmd.out.Error.Fatalf("No machine named '%s' exists. Did you run 'rig start --name=\"%s\"'?", cmd.machine.Name, cmd.machine.Name)
		}
	}

	// 1.5 Ensure docker / machine is running
	if !util.SupportsNativeDocker() {
		if !cmd.machine.IsRunning() {
			cmd.out.Error.Fatalf("Machine '%s' is not running. You may need to run 'rig start'", cmd.machine.Name)
		} else {
			cmd.out.Info.Printf("Docker Machine (%s) is running", cmd.machine.Name)
		}
	} else {
		if err := exec.Command("docker", "version").Run(); err != nil {
			cmd.out.Error.Fatalf("Docker is not running. You may need to run 'systemctl start docker'")
		} else {
			cmd.out.Info.Println("Docker is running")
		}
	}

	// 2. Check Docker API Version compatibility
	clientAPIVersion := util.GetDockerClientAPIVersion()
	serverAPIVersion, err := util.GetDockerServerAPIVersion()
	serverMinAPIVersion, _ := util.GetDockerServerMinAPIVersion()

	// Older clients can talk to newer servers, and when you ask a newer server
	// it's version in the presence of an older server it will downgrade it's
	// compatability as far as possible. So as long as the client API is not greater
	// than the servers current version or less than the servers minimum api version
	// then we are compatible
	constraintString := fmt.Sprintf("<= %s", serverAPIVersion)
	if serverMinAPIVersion != nil {
		constraintString = fmt.Sprintf(">= %s", serverMinAPIVersion)
	}
	apiConstraint, _ := version.NewConstraint(constraintString)

	if err != nil {
		cmd.out.Error.Println("Could not determine Docker versions: ", err)
	} else if clientAPIVersion.Equal(serverAPIVersion) {
		cmd.out.Info.Printf("Docker Client (%s) and Server (%s) have equal API Versions", clientAPIVersion, serverAPIVersion)
	} else if apiConstraint.Check(clientAPIVersion) {
		cmd.out.Info.Printf("Docker Client (%s) has Server compatible API version (%s). Server current (%s), Server min compat (%s)", clientAPIVersion, constraintString, serverAPIVersion, serverMinAPIVersion)
	} else {
		cmd.out.Error.Printf("Docker Client (%s) is incompatible with Server. Server current (%s), Server min compat (%s). Use `rig upgrade` to fix this.", clientAPIVersion, serverAPIVersion, serverMinAPIVersion)
	}

	// 3. Pull down the data from DNSDock. This will confirm we can resolve names as well
	//    as route to the appropriate IP addresses via the added route commands
	dnsRecords := DNSRecords{cmd.BaseCommand}
	if records, err := dnsRecords.LoadRecords(); err == nil {
		resolved := false
		for _, record := range records {
			if record["Name"] == "dnsdock" {
				resolved = true
				cmd.out.Info.Printf("DNS and routing services are working. DNSDock resolves to %s", record["IPs"])
				break
			}
		}

		if !resolved {
			cmd.out.Error.Println("Unable to verify DNS services are working.")
		}
	} else {
		cmd.out.Error.Println("Unable to verify DNS services and routing are working.")
		cmd.out.Error.Println(err)
	}

	// 4. Ensure that docker-machine-nfs script is available for our NFS mounts (Mac ONLY)
	if util.IsMac() {
		if err := util.Command("which", "docker-machine-nfs").Run(); err != nil {
			cmd.out.Error.Println("Docker Machine NFS is not installed.")
		} else {
			cmd.out.Info.Println("Docker Machine NFS is installed.")
		}
	}

	// 5. Check for storage on VM volume
	if !util.SupportsNativeDocker() {
		output, err := util.Command("docker-machine", "ssh", cmd.machine.Name, "df -h 2> /dev/null | grep /dev/sda1 | head -1 | awk '{print $5}' | sed 's/%//'").Output()
		if err == nil {
			dataUsage := strings.TrimSpace(string(output))
			if i, e := strconv.Atoi(dataUsage); e == nil {
				if i >= 85 && i < 95 {
					cmd.out.Warning.Printf("Data volume (/data) is %d%% used. Please free up space soon.", i)
				} else if i >= 95 {
					cmd.out.Error.Printf("Data volume (/data) is %d%% used. Please free up space. Try 'docker system prune' or removing old projects / databases from /data.", i)
				} else {
					cmd.out.Info.Printf("Data volume (/data) is %d%% used.", i)
				}
			} else {
				cmd.out.Warning.Printf("Unable to determine usage level of /data volume. Failed to parse '%s'", dataUsage)
			}
		} else {
			cmd.out.Warning.Printf("Unable to determine usage level of /data volume. Failed to execute 'df': %v", err)
		}
	}

	// 6. Check for storage on /Users
	if !util.SupportsNativeDocker() {
		output, err := util.Command("docker-machine", "ssh", cmd.machine.Name, "df -h 2> /dev/null | grep /Users | head -1 | awk '{print $5}' | sed 's/%//'").Output()
		if err == nil {
			userUsage := strings.TrimSpace(string(output))
			if i, e := strconv.Atoi(userUsage); e == nil {
				if i >= 85 && i < 95 {
					cmd.out.Warning.Printf("Root drive (/Users) is %d%% used. Please free up space soon.", i)
				} else if i >= 95 {
					cmd.out.Error.Printf("Root drive (/Users) is %d%% used. Please free up space.", i)
				} else {
					cmd.out.Info.Printf("Root drive (/Users) is %d%% used.", i)
				}
			} else {
				cmd.out.Warning.Printf("Unable to determine usage level of root drive (/Users). Failed to parse '%s'", userUsage)
			}
		} else {
			cmd.out.Warning.Printf("Unable to determine usage level of root drive (/Users). Failed to execute 'df': %v", err)
		}
	}

	return nil
}
