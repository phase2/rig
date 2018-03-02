package commands

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"

	"github.com/phase2/rig/util"
)

// ProjectSync is the command volume and file sync operations
type ProjectSync struct {
	BaseCommand
	Config *ProjectConfig
}

// ComposeFile is a minimal compose file struct to discover volumes
type ComposeFile struct {
	Volumes map[string]Volume
}

// Volume is a minimal volume spec to determine if a defined volume is declared external
type Volume struct {
	External bool
}

const unisonPort = 5000
const maxWatches = "100000"

// Commands returns the operations supported by this command
func (cmd *ProjectSync) Commands() []cli.Command {
	start := cli.Command{
		Name:        "sync:start",
		Aliases:     []string{"sync"},
		Usage:       "Start a Unison sync on local project directory.",
		Description: "Volume name will be discovered in the following order: outrigger project config > docker-compose file > current directory name",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "initial-sync-timeout",
				Value:  120,
				Usage:  "Maximum amount of time in seconds to allow for detecting each of start of the Unison container and start of initial sync. If you encounter failures detecting initial sync increasing this value may help. Search for sync on http://docs.outrigger.sh/faq/troubleshooting/ (not needed on linux)",
				EnvVar: "RIG_PROJECT_SYNC_TIMEOUT",
			},
			// Arbitrary sleep length but anything less than 3 wasn't catching
			// ongoing very quick file updates during a test
			cli.IntFlag{
				Name:   "initial-sync-wait",
				Value:  5,
				Usage:  "Time in seconds to wait between checks to see if initial sync has finished. (not needed on linux)",
				EnvVar: "RIG_PROJECT_INITIAL_SYNC_WAIT",
			},
			// Override the local sync path.
			cli.StringFlag{
				Name:  "dir",
				Value: "",
				Usage: "Specify the location in the local filesystem to be synced. If not used it will look for the directory of project configuration or fall back to current working directory. Use '--dir=.' to guarantee current working directory is used.",
			},
		},
		Before: cmd.Before,
		Action: cmd.RunStart,
	}
	stop := cli.Command{
		Name:        "sync:stop",
		Usage:       "Stops a Unison sync on local project directory.",
		Description: "Volume name will be discovered in the following order: outrigger project config > docker-compose file > current directory name",
		Flags: []cli.Flag{
			// Override the local sync path.
			cli.StringFlag{
				Name:  "dir",
				Value: "",
				Usage: "Specify the location in the local filesystem to be synced. If not used it will look for the directory of project configuration or fall back to current working directory. Use '--dir=.' to guarantee current working directory is used.",
			},
		},
		Before: cmd.Before,
		Action: cmd.RunStop,
	}

	return []cli.Command{start, stop}
}

// RunStart executes the `rig project sync:start` command to start the Unison sync process.
func (cmd *ProjectSync) RunStart(ctx *cli.Context) error {
	cmd.Config = NewProjectConfig()
	if cmd.Config.NotEmpty() {
		cmd.out.Verbose("Loaded project configuration from %s", cmd.Config.Path)
	}

	// Determine the working directory for CWD-sensitive operations.
	var workingDir, err = cmd.DeriveLocalSyncPath(cmd.Config, ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}

	// Determine the volume name to be used across all operating systems.
	// For cross-compatibility the way this volume is set up will vary.
	volumeName := cmd.GetVolumeName(cmd.Config, workingDir)

	switch platform := runtime.GOOS; platform {
	case "linux":
		cmd.out.Verbose("Setting up local volume: %s", volumeName)
		return cmd.SetupBindVolume(volumeName, workingDir)
	default:
		cmd.out.Verbose("Starting sync with volume: %s", volumeName)
		return cmd.StartUnisonSync(ctx, volumeName, cmd.Config, workingDir)
	}
}

// StartUnisonSync will create and launch the volumes and containers on systems that need/support Unison
func (cmd *ProjectSync) StartUnisonSync(ctx *cli.Context, volumeName string, config *ProjectConfig, workingDir string) error {
	cmd.out.Spin("Starting Outrigger Filesync (unison)...")

	// Ensure the processes can handle a large number of watches
	if err := cmd.machine.SetSysctl("fs.inotify.max_user_watches", maxWatches); err != nil {
		cmd.Failure(fmt.Sprintf("Failure configuring file watches on Docker Machine: %v", err), "INOTIFY-WATCH-FAILURE", 12)
	}

	cmd.out.SpinWithVerbose("Starting sync volume: %s", volumeName)
	if err := util.Command("docker", "volume", "create", volumeName).Run(); err != nil {
		return cmd.Failure(fmt.Sprintf("Failed to create sync volume: %s", volumeName), "VOLUME-CREATE-FAILED", 13)
	}
	cmd.out.Info("Sync volume '%s' created", volumeName)
	cmd.out.SpinWithVerbose(fmt.Sprintf("Starting sync container: %s (same name)", volumeName))
	unisonMinorVersion := cmd.GetUnisonMinorVersion()

	cmd.out.Verbose("Local Unison version for compatibilty: %s", unisonMinorVersion)
	util.Command("docker", "container", "stop", volumeName).Run()
	containerArgs := []string{
		"container", "run", "--detach", "--rm",
		"-v", fmt.Sprintf("%s:/unison", volumeName),
		"-e", "UNISON_DIR=/unison",
		"-l", fmt.Sprintf("com.dnsdock.name=%s", volumeName),
		"-l", "com.dnsdock.image=volume.outrigger",
		"--name", volumeName,
		fmt.Sprintf("outrigger/unison:%s", unisonMinorVersion),
	}
	if err := util.Command("docker", containerArgs...).Run(); err != nil {
		cmd.Failure(fmt.Sprintf("Failure starting sync container %s: %v", volumeName, err), "SYNC-CONTAINER-START-FAILED", 13)
	}

	ip, err := cmd.WaitForUnisonContainer(volumeName, ctx.Int("initial-sync-timeout"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-INIT-FAILED", 13)
	}
	cmd.out.Info("Sync container '%s' started", volumeName)
	cmd.out.SpinWithVerbose("Initializing file sync...")

	// Determine the location of the local Unison log file.
	var logFile = fmt.Sprintf("%s.log", volumeName)
	// Remove the log file, the existence of the log file will mean that sync is
	// up and running. If the logfile does not exist, do not complain. If the
	// filesystem cannot delete the file when it exists, it will lead to errors.
	if err := util.RemoveFile(logFile, workingDir); err != nil {
		cmd.out.Verbose("Could not remove Unison log file: %s: %s", logFile, err.Error())
	}

	// Initiate local Unison process.
	unisonArgs := []string{
		".",
		fmt.Sprintf("socket://%s:%d/", ip, unisonPort),
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

	/* #nosec */
	command := exec.Command("unison", unisonArgs...)
	command.Dir = workingDir
	cmd.out.Verbose("Sync execution - Working Directory: %s", workingDir)
	if err = util.Convert(command).Start(); err != nil {
		return cmd.Failure(fmt.Sprintf("Failure starting local Unison process: %v", err), "UNISON-START-FAILED", 13)
	}

	if err := cmd.WaitForSyncInit(logFile, workingDir, ctx.Int("initial-sync-timeout"), ctx.Int("initial-sync-wait")); err != nil {
		return cmd.Failure(err.Error(), "UNISON-SYNC-FAILED", 13)
	}

	cmd.out.Info("Watch unison process activities in the sync log: %s", logFile)

	return cmd.Success("Unison sync started successfully")
}

// SetupBindVolume will create minimal Docker Volumes for systems that have native container/volume support
func (cmd *ProjectSync) SetupBindVolume(volumeName string, workingDir string) error {
	cmd.out.SpinWithVerbose("Starting local bind volume: %s", volumeName)
	util.Command("docker", "volume", "rm", volumeName).Run()

	volumeArgs := []string{
		"volume", "create",
		"--opt", "type=none",
		"--opt", fmt.Sprintf("device=%s", workingDir),
		"--opt", "o=bind",
		volumeName,
	}

	if err := util.Command("docker", volumeArgs...).Run(); err != nil {
		return cmd.Failure(err.Error(), "BIND-VOLUME-FAILURE", 13)
	}

	return cmd.Success("Bind volume created")
}

// RunStop executes the `rig project sync:stop` command to shut down and unison containers
func (cmd *ProjectSync) RunStop(ctx *cli.Context) error {
	if runtime.GOOS == "linux" {
		return cmd.Success("No Unison container to stop, using local bind volume")
	}
	cmd.out.Spin(fmt.Sprintf("Stopping Unison container"))

	cmd.Config = NewProjectConfig()
	if cmd.Config.NotEmpty() {
		cmd.out.Verbose("Loaded project configuration from %s", cmd.Config.Path)
	}

	// Determine the working directory for CWD-sensitive operations.
	var workingDir, err = cmd.DeriveLocalSyncPath(cmd.Config, ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}

	volumeName := cmd.GetVolumeName(cmd.Config, workingDir)
	cmd.out.Spin(fmt.Sprintf("Stopping Unison container (%s)", volumeName))
	if err := util.Command("docker", "container", "stop", volumeName).Run(); err != nil {
		return cmd.Failure(err.Error(), "SYNC-CONTAINER-FAILURE", 13)
	}

	return cmd.Success(fmt.Sprintf("Unison container '%s' stopped", volumeName))
}

// GetVolumeName will find the volume name through a variety of fall backs
func (cmd *ProjectSync) GetVolumeName(config *ProjectConfig, workingDir string) string {
	// 1. Check for config
	if config.Sync != nil && config.Sync.Volume != "" {
		return config.Sync.Volume
	}

	// 2. Parse compose file looking for an external volume named *-sync
	if composeConfig, err := cmd.LoadComposeFile(); err == nil {
		for name, volume := range composeConfig.Volumes {
			if strings.HasSuffix(name, "-sync") && volume.External {
				return name
			}
		}
	}

	// 3. Use local dir for the volume name
	var _, folder = path.Split(workingDir)
	return fmt.Sprintf("%s-sync", folder)
}

// LoadComposeFile will load the proper compose file
func (cmd *ProjectSync) LoadComposeFile() (*ComposeFile, error) {
	yamlFile, err := ioutil.ReadFile("./docker-compose.yml")

	if err == nil {
		var config ComposeFile
		if e := yaml.Unmarshal(yamlFile, &config); e != nil {
			cmd.out.Channel.Error.Fatalf("YAML Parsing Failure: %s", e)
		}
		return &config, nil
	}

	return nil, err
}

// WaitForUnisonContainer will wait for the unison container port to allow connections
// Due to the fact that we don't compile with -cgo (so we can build using Docker),
// we need to discover the IP address of the container instead of using the DNS name
// when compiled without -cgo this executable will not use the native mac dns resolution
// which is how we have configured dnsdock to provide names for containers.
func (cmd *ProjectSync) WaitForUnisonContainer(containerName string, timeoutSeconds int) (string, error) {
	cmd.out.SpinWithVerbose("Sync container '%s' started , waiting for unison server process...", containerName)

	var timeoutLoopSleep = time.Duration(100) * time.Millisecond
	// * 10 here because we loop once every 100 ms and we want to get to seconds
	var timeoutLoops = timeoutSeconds * 10

	output, err := util.Command("docker", "inspect", "--format", "{{.NetworkSettings.IPAddress}}", containerName).Output()
	if err != nil {
		return "", fmt.Errorf("error inspecting sync container %s: %v", containerName, err)
	}
	ip := strings.Trim(string(output), "\n")

	cmd.out.Verbose("Checking for Unison network connection on %s %d", ip, unisonPort)
	for i := 1; i <= timeoutLoops; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, unisonPort))
		if err == nil {
			conn.Close()
			return ip, nil
		}

		cmd.out.SpinWithVerbose("Failure: %v", err)
		time.Sleep(timeoutLoopSleep)
	}

	return "", fmt.Errorf("sync container %s failed to start", containerName)
}

// WaitForSyncInit will wait for the local unison process to finish initializing
// when the log file exists and has stopped growing in size
func (cmd *ProjectSync) WaitForSyncInit(logFile string, workingDir string, timeoutSeconds int, syncWaitSeconds int) error {
	cmd.out.SpinWithVerbose("Waiting for initial sync detection...")

	// The use of os.Stat below is not subject to our working directory configuration,
	// so to ensure we can stat the log file we convert it to an absolute path.
	if logFilePath, err := util.AbsJoin(workingDir, logFile); err != nil {
		cmd.out.Error(err.Error())
	} else {
		// Create a temp file to cause a sync action
		var tempFile = ".rig-check-sync-start"

		if err := util.TouchFile(tempFile, workingDir); err != nil {
			cmd.out.Channel.Error.Fatal("Could not create file used to detect initial sync: %s", err.Error())
		}
		cmd.out.Verbose("Creating temporary file so we can watch for Unison initialization: %s", tempFile)

		var timeoutLoopSleep = time.Duration(100) * time.Millisecond
		// * 10 here because we loop once every 100 ms and we want to get to seconds
		var timeoutLoops = timeoutSeconds * 10
		var statSleep = time.Duration(syncWaitSeconds) * time.Second
		for i := 1; i <= timeoutLoops; i++ {
			statInfo, err := os.Stat(logFilePath)
			if err == nil {
				cmd.out.Info("Initial sync detected")

				cmd.out.SpinWithVerbose("Waiting for initial sync to finish")
				// Initialize at -2 to force at least one loop
				var lastSize = int64(-2)
				for lastSize != statInfo.Size() {
					time.Sleep(statSleep)
					lastSize = statInfo.Size()
					if statInfo, err = os.Stat(logFilePath); err != nil {
						cmd.out.Error(err.Error())
						lastSize = -1
					}
				}

				// Remove the temp file, waiting until after sync so spurious
				// failure message doesn't show in log
				if err := util.RemoveFile(tempFile, workingDir); err != nil {
					cmd.out.Warning("Could not remove the temporary file: %s: %s", tempFile, err.Error())
				}

				cmd.out.Info("File sync completed")
				return nil
			}

			time.Sleep(timeoutLoopSleep)
		}

		// The log file was not created, the sync has not started yet
		if err := util.RemoveFile(tempFile, workingDir); err != nil {
			// While the removal of the tempFile is not significant, if something
			// prevents removal there may be a bigger problem.
			cmd.out.Warning("Could not remove the temporary file: %s", err.Error())
		}
	}

	cmd.out.Error("Initial sync detection failed, this could indicate a need to increase the initial-sync-timeout. See rig project sync --help")
	return fmt.Errorf("Failed to detect start of initial sync")
}

// GetUnisonMinorVersion will return the local Unison version to try to load a compatible unison image
// This function discovers a semver like 2.48.4 and return 2.48
func (cmd *ProjectSync) GetUnisonMinorVersion() string {
	output, _ := util.Command("unison", "-version").Output()
	re := regexp.MustCompile(`unison version (\d+\.\d+\.\d+)`)
	rawVersion := re.FindAllStringSubmatch(string(output), -1)[0][1]
	v := version.Must(version.NewVersion(rawVersion))
	segments := v.Segments()
	return fmt.Sprintf("%d.%d", segments[0], segments[1])
}

// DeriveLocalSyncPath will derive the source path for the local host side of the file sync.
// If there is no override, use an empty string.
func (cmd *ProjectSync) DeriveLocalSyncPath(config *ProjectConfig, override string) (string, error) {
	var workingDir string
	if override != "" {
		workingDir = override
	} else if config.NotEmpty() {
		workingDir = filepath.Dir(config.Path)
	} else if cwd, err := os.Getwd(); err == nil {
		workingDir = cwd
	} else {
		return "", fmt.Errorf("Could not identify a source directory for file sync")
	}

	absoluteWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("Could not process the directory into an absolute file path: %s", workingDir)
	}

	if _, err := os.Stat(absoluteWorkingDir); !os.IsNotExist(err) {
		return absoluteWorkingDir, nil
	}

	return "", fmt.Errorf("Identified sync source path does not exist: %s", absoluteWorkingDir)
}
