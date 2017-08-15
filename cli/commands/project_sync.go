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
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "initial-sync-timeout",
				Value:  60,
				Usage:  "Maximum amount of time in seconds to allow for detecting each of start of the unison container and start of initial sync",
				EnvVar: "RIG_PROJECT_SYNC_TIMEOUT",
			},
			cli.IntFlag{
				Name:   "initial-sync-wait",
				Value:  5,
				Usage:  "Time in seconds to wait between checks to see if initial sync has finished.",
				EnvVar: "RIG_PROJECT_INITIAL_SYNC_WAIT",
			},
		},
		Before: cmd.Before,
		Action: cmd.RunStart,
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
	config := NewProjectConfig()
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

	var ip = cmd.WaitForUnisonContainer(volumeName, ctx.Int("initial-sync-timeout"))

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
		"-ignore", fmt.Sprintf("Name %s", logFile),
	}
	// Append ProjectConfig ignores here
	if config.Sync != nil {
		for _, ignore := range config.Sync.Ignore {
			unisonArgs = append(unisonArgs, "-ignore", ignore)
		}
	}

	cmd.out.Verbose.Printf("Unison Args: %s", strings.Join(unisonArgs[:], " "))
	if err = exec.Command("unison", unisonArgs...).Start(); err != nil {
		cmd.out.Error.Fatalf("Error starting local unison process: %v", err)
	}

	cmd.WaitForSyncInit(logFile, ctx.Int("initial-sync-timeout"), ctx.Int("initial-sync-wait"))

	return nil
}

// Start the unison sync process
func (cmd *ProjectSync) RunStop(ctx *cli.Context) error {
	config := NewProjectConfig()
	volumeName := cmd.GetVolumeName(ctx, config)
	cmd.out.Verbose.Printf("Stopping sync with volume: %s", volumeName)

	cmd.out.Info.Println("Stopping unison container")
	exec.Command("docker", "container", "stop", volumeName).Run()

	return nil
}

// Find the volume name through a variety of fall backs
func (cmd *ProjectSync) GetVolumeName(ctx *cli.Context, config *ProjectConfig) string {
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
		return fmt.Sprintf("%s-sync", folder)
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
func (cmd *ProjectSync) WaitForUnisonContainer(containerName string, timeoutSeconds int) string {
	cmd.out.Info.Println("Waiting for container to start")

	var timeoutLoopSleep = time.Duration(100) * time.Millisecond
	// * 10 here because we loop once every 100 ms and we want to get to seconds
	var timeoutLoops = timeoutSeconds * 10

	output, err := exec.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", containerName).Output()
	if err != nil {
		cmd.out.Error.Fatalf("Error inspecting sync container %s: %v", containerName, err)
	}
	ip := strings.Trim(string(output), "\n")

	cmd.out.Verbose.Printf("Checking for unison network connection on %s %d", ip, UNISON_PORT)
	for i := 1; i <= timeoutLoops; i++ {
		if conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, UNISON_PORT)); err == nil {
			conn.Close()
			return ip
		} else {
			cmd.out.Info.Printf("Error: %v", err)
			time.Sleep(timeoutLoopSleep)
		}
	}
	cmd.out.Error.Fatal("Sync container failed to start!")
	return ""
}

// The local unison process is finished initializing when the log file exists
// and has stopped growing in size
func (cmd *ProjectSync) WaitForSyncInit(logFile string, timeoutSeconds int, syncWaitSeconds int) {
	cmd.out.Info.Print("Waiting for initial sync detection")

	var tempFile = fmt.Sprintf(".%s.tmp", logFile)
	var timeoutLoopSleep = time.Duration(100) * time.Millisecond
	// * 10 here because we loop once every 100 ms and we want to get to seconds
	var timeoutLoops = timeoutSeconds * 10

	// Create a temp file to cause a sync action
	exec.Command("touch", tempFile).Run()

	for i := 1; i <= timeoutLoops; i++ {
		if i%10 == 0 {
			os.Stdout.WriteString(".")
		}
		if statInfo, err := os.Stat(logFile); err == nil {
			os.Stdout.WriteString(" initial sync detected\n")
			// Remove the temp file now that we are running
			exec.Command("rm", "-f", tempFile).Run()

			cmd.out.Info.Print("Waiting for initial sync to finish")
			// Arbitrary sleep length but anything less than 3 wasn't catching
			// ongoing very quick file updates during a test
			var statSleep = time.Duration(syncWaitSeconds) * time.Second
			// Initialize at -2 to force at least one loop
			var lastSize = int64(-2)
			for lastSize != statInfo.Size() {
				os.Stdout.WriteString(".")
				time.Sleep(statSleep)
				lastSize = statInfo.Size()
				if statInfo, err = os.Stat(logFile); err != nil {
					cmd.out.Info.Print(err.Error())
					lastSize = -1
				}
			}
			os.Stdout.WriteString(" done\n")
			return
		} else {
			time.Sleep(timeoutLoopSleep)
		}
	}

	// The log file was not created, the sync has not started yet
	exec.Command("rm", "-f", tempFile).Run()
	cmd.out.Error.Fatal("Failed to detect start of initial sync! Check sync log file.")
}

// Get the local Unison version to try to load a compatible unison image
// This function discovers a semver like 2.48.4 and return 2.48
func (cmd *ProjectSync) GetUnisonMinorVersion() string {
	output, _ := exec.Command("unison", "-version").Output()
	re := regexp.MustCompile("unison version (\\d+\\.\\d+\\.\\d+)")
	rawVersion := re.FindAllStringSubmatch(string(output), -1)[0][1]
	v := version.Must(version.NewVersion(rawVersion))
	segments := v.Segments()
	return fmt.Sprintf("%d.%d", segments[0], segments[1])
}
