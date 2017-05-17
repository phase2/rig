package commands

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/phase2/rig/cli/commands/project"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

type ProjectSync struct {
	BaseCommand
}

// Minimal compose file struct to discover volumes
type ComposeFile struct {
	Volumes map[string]Volume
}

type Volume struct {
	External bool
}

const UNISON_PORT = 5000
const MAX_WATCHES = "100000"

func (cmd *ProjectSync) Commands() []cli.Command {
	start := cli.Command{
		Name:        "sync:start",
		Aliases:     []string{"sync"},
		Usage:       "Start a unison sync on local project directory. Optionally provide a volume name.",
		ArgsUsage:   "[optional volume name]",
		Description: "Volume name will be discovered in the following order: argument to this command > outrigger project config > docker-compose file > current directory name",
		Before:      cmd.Before,
		Action:      cmd.RunStart,
	}
	stop := cli.Command{
		Name:        "sync:stop",
		Usage:       "Stops a unison sync on local project directory. Optionally provide a volume name.",
		ArgsUsage:   "[optional volume name]",
		Description: "Volume name will be discovered in the following order: argument to this command > outrigger project config > docker-compose file > current directory name",
		Before:      cmd.Before,
		Action:      cmd.RunStop,
	}

	return []cli.Command{start, stop}
}

// Start the unison sync process
func (cmd *ProjectSync) RunStart(ctx *cli.Context) error {
	project.ConfigInit()
	config := project.GetProjectConfigFromFile(project.GetConfigPath())
	volumeName := cmd.GetVolumeName(ctx, config)
	cmd.out.Verbose.Printf("Starting sync with volume: %s", volumeName)

	// Ensure the processes can handle a large number of watches
	if err := cmd.machine.SetSysctl("fs.inotify.max_user_watches", MAX_WATCHES); err != nil {
		cmd.out.Error.Fatalf("Error configuring file watches on Docker Machine: %v", err)
	}

	cmd.out.Info.Printf("Starting sync volume: %s", volumeName)
	exec.Command("docker", "volume", "create", volumeName).Run()

	cmd.out.Info.Println("Starting unison container")
	unisonMinorVersion := cmd.GetUnisonMinorVersion()
	cmd.out.Verbose.Printf("Local unison version for compatibilty: %s", unisonMinorVersion)
	exec.Command("docker", "container", "stop", volumeName).Run()
	err := exec.Command("docker", "container", "run", "--detach", "--rm",
		"-v", fmt.Sprintf("%s:/unison", volumeName),
		"-e", "UNISON_DIR=/unison",
		"-l", fmt.Sprintf("com.dnsdock.name=%s", volumeName),
		"-l", "com.dnsdock.image=volume.outrigger",
		"--name", volumeName,
		fmt.Sprintf("outrigger/unison:%s", unisonMinorVersion),
	).Run()
	if err != nil {
		cmd.out.Error.Fatalf("Error starting sync container %s: %v", volumeName, err)
	}

	var ip = cmd.WaitForUnisonContainer(volumeName)

	cmd.out.Info.Println("Initializing sync")

	// Remove the log file, the existence of the log file will mean that sync is up and running
	var logFile = fmt.Sprintf("%s.log", volumeName)
	exec.Command("rm", "-f", logFile).Run()

	unisonArgs := []string{
		".",
		fmt.Sprintf("socket://%s:%d/", ip, UNISON_PORT),
		"-auto", "-batch", "-silent", "-contactquietly",
		"-repeat", "watch",
		"-prefer", ".",
		"-logfile", logFile,
		"-ignore", "Name .git",
		"-ignore", fmt.Sprintf("Name %s", logFile),
	}
	// Append ProjectConfig ignores here
	for _, ignore := range config.Sync.Ignore {
		unisonArgs = append(unisonArgs, "-ignore", ignore)
	}

	cmd.out.Verbose.Printf("Unison Args: %s", strings.Join(unisonArgs[:], " "))
	if err = exec.Command("unison", unisonArgs...).Start(); err != nil {
		cmd.out.Error.Fatalf("Error starting local unison process: %v", err)
	}

	cmd.WaitForSyncInit(logFile)

	return nil
}

// Start the unison sync process
func (cmd *ProjectSync) RunStop(ctx *cli.Context) error {
	project.ConfigInit()
	config := project.GetProjectConfigFromFile(project.GetConfigPath())
	volumeName := cmd.GetVolumeName(ctx, config)
	cmd.out.Verbose.Printf("Stopping sync with volume: %s", volumeName)

	cmd.out.Info.Println("Stopping unison container")
	exec.Command("docker", "container", "stop", volumeName).Run()

	return nil
}

// Find the volume name through a variety of fall backs
func (cmd *ProjectSync) GetVolumeName(ctx *cli.Context, config project.ProjectConfig) string {
	// 1. Check for argument
	if ctx.Args().Present() {
		return ctx.Args().First()
	}

	// 2. Check for config
	if config.Sync != nil && config.Sync.Volume != "" {
		return config.Sync.Volume
	}

	// 3. Parse compose file looking for an external volume named *-sync
	if composeConfig, err := cmd.LoadComposeFile(); err == nil {
		for name, volume := range composeConfig.Volumes {
			if strings.HasSuffix(name, "-sync") && volume.External {
				return name
			}
		}
	}

	// 4. Use local dir for the volume name
	if dir, err := os.Getwd(); err == nil {
		var _, folder = path.Split(dir)
		return folder
	} else {
		cmd.out.Error.Println(err)
	}

	cmd.out.Error.Fatal("Unable to determine a name for the sync volume")
	return ""
}

// Load the proper compose file
func (cmd *ProjectSync) LoadComposeFile() (*ComposeFile, error) {
	yamlFile, err := ioutil.ReadFile("./docker-compose.yml")

	if err == nil {
		var config ComposeFile
		if err := yaml.Unmarshal(yamlFile, &config); err != nil {
			cmd.out.Error.Fatalf("YAML Parsing Error: %s", err)
		}
		return &config, nil
	}

	return nil, err
}

// Wait for the unison container port to allow connections
// Due to the fact that we don't compile with -cgo (so we can build using Docker),
// we need to discover the IP address of the container instead of using the DNS name
// when compiled without -cgo this executable will not use the native mac dns resolution
// which is how we have configured dnsdock to provide names for containers.
func (cmd *ProjectSync) WaitForUnisonContainer(containerName string) string {
	cmd.out.Info.Println("Waiting for container to start")

	output, err := exec.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", containerName).Output()
	if err != nil {
		cmd.out.Error.Fatalf("Error inspecting sync container %s: %v", containerName, err)
	}
	ip := strings.Trim(string(output), "\n")

	cmd.out.Verbose.Printf("Checking for unison network connection on %s %d", ip, UNISON_PORT)
	for i := 1; i <= 100; i++ {
		if conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, UNISON_PORT)); err == nil {
			defer conn.Close()
			return ip
		} else {
			cmd.out.Info.Printf("Error: %v", err)
			time.Sleep(time.Duration(100) * time.Millisecond)
		}
	}
	cmd.out.Error.Fatal("Sync container failed to start!")
	return ""
}

// The local unison process is finished initializing when the log file exists
func (cmd *ProjectSync) WaitForSyncInit(logFile string) {
	cmd.out.Info.Print("Waiting for initial sync to finish...")

	var tempFile = fmt.Sprintf(".%s.tmp", logFile)

	// Create a temp file to cause a sync action
	exec.Command("touch", tempFile).Run()

	// Lets check for 60 seconds, while waiting for initial sync to complete
	for i := 1; i <= 600; i++ {
		if i%10 == 0 {
			os.Stdout.WriteString(".")
		}
		if _, err := os.Stat(logFile); err == nil {
			// Remove the temp file now that we are running
			os.Stdout.WriteString("done\n")
			exec.Command("rm", "-f", tempFile).Run()
			return
		} else {
			time.Sleep(time.Duration(100) * time.Millisecond)
		}
	}

	// The log file was not created, the sync has not started yet
	exec.Command("rm", "-f", tempFile).Run()
	cmd.out.Error.Fatal("Sync container failed to start!")
}

// Get the local Unison version to try to load a compatible unison image
// This function discovers a semver like 2.48.4 and return 2.48
func (cmd *ProjectSync) GetUnisonMinorVersion() string {
	output, _ := exec.Command("unison", "-version").Output()
	re := regexp.MustCompile("unison version (\\d+\\.\\d+\\.\\d+)")
	rawVersion := re.FindAllStringSubmatch(string(output), -1)[0][1]
	version := version.Must(version.NewVersion(rawVersion))
	segments := version.Segments()
	return fmt.Sprintf("%d.%d", segments[0], segments[1])
}
