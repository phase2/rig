package commands

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
		Category:    "File Sync",
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
		Category:    "File Sync",
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
	name := cli.Command{
		Name:        "sync:name",
		Category:    "File Sync",
		Usage:       "Retrieves the name used for the Unison volume and container.",
		Description: "This will perform the same name discovery used by sync:start and returns it to ease scripting.",
		Flags: []cli.Flag{
			// Override the local sync path.
			cli.StringFlag{
				Name:  "dir",
				Value: "",
				Usage: "Specify the location in the local filesystem to be synced. If not used it will look for the directory of project configuration or fall back to current working directory. Use '--dir=.' to guarantee current working directory is used.",
			},
		},
		Before: cmd.Before,
		Action: cmd.RunName,
	}
	check := cli.Command{
		Name:        "sync:check",
		Category:    "File Sync",
		Usage:       "Run doctor checks on the state of your unison file sync.",
		Description: "This is intended to facilitate easy verification whether the filesync is down.",
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
		Action: cmd.RunCheck,
	}
	purge := cli.Command{
		Name:        "sync:purge",
		Category:    "File Sync",
		Usage:       "Purges an existing sync volume for the current project/directory.",
		Description: "This goes beyond sync:stop to remove the Docker plumbing of the file sync for a clean restart.",
		Flags: []cli.Flag{
			// Override the local sync path.
			cli.StringFlag{
				Name:  "dir",
				Value: "",
				Usage: "Specify the location in the local filesystem to be synced. If not used it will look for the directory of project configuration or fall back to current working directory. Use '--dir=.' to guarantee current working directory is used.",
			},
		},
		Before: cmd.Before,
		Action: cmd.RunPurge,
	}
	return []cli.Command{start, stop, name, check, purge}
}

// RunStart executes the `rig project sync:start` command to start the Unison sync process.
func (cmd *ProjectSync) RunStart(ctx *cli.Context) error {
	volumeName, workingDir, err := cmd.initializeSettings(ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}

	switch platform := runtime.GOOS; platform {
	case "linux":
		cmd.out.Verbose("Setting up local volume: %s", volumeName)
		return cmd.SetupBindVolume(volumeName, workingDir)
	default:
		cmd.out.Verbose("Starting sync with volume: %s", volumeName)
		return cmd.StartUnisonSync(ctx, volumeName, cmd.Config, workingDir)
	}
}

// RunStop executes the `rig project sync:stop` command to shut down and unison containers
func (cmd *ProjectSync) RunStop(ctx *cli.Context) error {
	if util.IsLinux() {
		return cmd.Success("No Unison container to stop, using local bind volume")
	}
	cmd.out.Spin(fmt.Sprintf("Stopping Unison container"))

	volumeName, _, err := cmd.initializeSettings(ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}

	cmd.out.Spin(fmt.Sprintf("Stopping Unison container (%s)", volumeName))
	if err := util.Command("docker", "container", "stop", volumeName).Run(); err != nil {
		return cmd.Failure(err.Error(), "SYNC-CONTAINER-FAILURE", 13)
	}

	return cmd.Success(fmt.Sprintf("Unison container '%s' stopped", volumeName))
}

// RunName provides the name of the sync volume and container. This is made available to facilitate scripting.
func (cmd *ProjectSync) RunName(ctx *cli.Context) error {
	name, _, err := cmd.initializeSettings(ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}

	fmt.Println(name)
	return nil
}

// RunCheck performs a doctor-like examination of the file sync health.
func (cmd *ProjectSync) RunCheck(ctx *cli.Context) error {
	cmd.out.Spin("Preparing test of unison filesync...")
	volumeName, workingDir, err := cmd.initializeSettings(ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}
	cmd.out.Info("Ready to begin unison test")
	cmd.out.Spin("Checking for unison container...")
	if running := util.ContainerRunning(volumeName); !running {
		return cmd.Failure(fmt.Sprintf("Unison container (%s) is not running", volumeName), "SYNC-CHECK-FAILED", 13)
	}
	cmd.out.Info("Unison container found: %s", volumeName)
	cmd.out.Spin("Check unison container process is listening...")
	if _, err := cmd.WaitForUnisonContainer(volumeName, ctx.Int("initial-sync-timeout")); err != nil {
		cmd.out.Error("Unison process not listening")
		return cmd.Failure(err.Error(), "SYNC-CHECK-FAILED", 13)
	}
	cmd.out.Info("Unison process is listening")

	// Determine if sync progress can be tracked.
	//cmd.out.Spin("Syncing a test file...")
	cmd.out.Info("Preparing live file sync test")
	var logFile = cmd.LogFileName(volumeName)
	if err := cmd.WaitForSyncInit(logFile, workingDir, ctx.Int("initial-sync-timeout"), ctx.Int("initial-sync-wait")); err != nil {
		return cmd.Failure(err.Error(), "UNISON-SYNC-FAILED", 13)
	}

	// Sidestepping the notification so rig sync:check can be run as a background process.
	cmd.out.Info("Sync check completed successfully")
	return nil
}

// RunPurge cleans out the project sync state.
func (cmd *ProjectSync) RunPurge(ctx *cli.Context) error {
	if util.IsLinux() {
		return cmd.Success("No Unison process to clean up.")
	}

	volumeName, workingDir, err := cmd.initializeSettings(ctx.String("dir"))
	if err != nil {
		return cmd.Failure(err.Error(), "SYNC-PATH-ERROR", 12)
	}

	cmd.out.Spin("Checking for unison container...")
	if running := util.ContainerRunning(volumeName); running {
		cmd.out.Spin(fmt.Sprintf("Stopping Unison container (%s)", volumeName))
		if stopErr := util.Command("docker", "container", "stop", volumeName).Run(); stopErr != nil {
			cmd.out.Warn("Could not stop unison container (%s): Maybe it's already stopped?", volumeName)
		} else {
			cmd.out.Info("Stopped unison container (%s)", volumeName)
		}
	} else {
		cmd.out.Info("No running unison container.")
	}

	logFile := cmd.LogFileName(volumeName)
	cmd.out.Spin(fmt.Sprintf("Removing unison log file: %s", logFile))
	if util.FileExists(logFile, workingDir) {
		if removeErr := util.RemoveFile(logFile, workingDir); removeErr != nil {
			cmd.out.Error("Could not remove unison log file: %s: %s", logFile, removeErr.Error())
		} else {
			cmd.out.Info("Removed unison log file: %s", logFile)
		}
	} else {
		cmd.out.Info("Log file does not exist")
	}

	// Remove sync fragment files.
	cmd.out.Spin("Removing .unison directories")
	if removeGlobErr := util.RemoveFileGlob("*.unison*", workingDir, cmd.out); removeGlobErr != nil {
		cmd.out.Warning("Could not remove .unison directories: %s", removeGlobErr)
	} else {
		cmd.out.Info("Removed all .unison directories")
	}

	cmd.out.Spin(fmt.Sprintf("Removing sync volume: %s", volumeName))
	// @TODO capture the volume rm error text to display to user!
	out, rmErr := util.Command("docker", "volume", "rm", "--force", volumeName).CombinedOutput()
	if rmErr != nil {
		fmt.Println(rmErr.Error())
		return cmd.Failure(string(out), "SYNC-VOLUME-REMOVE-FAILURE", 13)
	}

	cmd.out.Info("Sync volume (%s) removed", volumeName)
	return nil
}

// initializeSettings pulls together the configuration and contextual settings
// used for all sync operations.
func (cmd *ProjectSync) initializeSettings(dir string) (string, string, error) {
	cmd.Config = NewProjectConfig()
	if cmd.Config.NotEmpty() {
		cmd.out.Verbose("Loaded project configuration from %s", cmd.Config.Path)
	}

	// Determine the working directory for CWD-sensitive operations.
	var workingDir, err = cmd.DeriveLocalSyncPath(cmd.Config, dir)
	if err != nil {
		return "", "", err
	}

	// Determine the volume name to be used across all operating systems.
	// For cross-compatibility the way this volume is set up will vary.
	volumeName := cmd.GetVolumeName(cmd.Config, workingDir)

	return volumeName, workingDir, nil
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
	unisonMinorVersion := util.GetUnisonMinorVersion()

	cmd.out.Verbose("Local Unison version for compatibility: %s", unisonMinorVersion)
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
	var logFile = cmd.LogFileName(volumeName)
	// Remove the log file, the existence of the log file will mean that sync is
	// up and running. If the logfile does not exist, do not complain. If the
	// filesystem cannot delete the file when it exists, it will lead to errors.
	if removeErr := util.RemoveFile(logFile, workingDir); removeErr != nil {
		cmd.out.Verbose("Could not remove Unison log file: %s: %s", logFile, removeErr.Error())
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

// LogFileName gets the unison sync file name.
// Be sure to convert it to an absolute path if used with functions that cannot
// use the working directory context.
func (cmd *ProjectSync) LogFileName(name string) string {
	return fmt.Sprintf("%s.log", name)
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
		cmd.out.Verbose("Attempt #%d...", i)
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ip, unisonPort))
		if err == nil {
			conn.Close()
			cmd.out.Verbose("Connected to unison on %s", containerName)
			return ip, nil
		}

		cmd.out.SpinWithVerbose("Failure: %v", err)
		time.Sleep(timeoutLoopSleep)
	}

	return "", fmt.Errorf("sync container %s is unreachable by unison", containerName)
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
			cmd.out.Channel.Error.Fatal(fmt.Sprintf("Could not create file used to detect initial sync: %s", err.Error()))
		}
		cmd.out.Verbose("Creating temporary file so we can watch for Unison initialization: %s", tempFile)

		var timeoutLoopSleep = time.Duration(100) * time.Millisecond
		// * 10 here because we loop once every 100 ms and we want to get to seconds
		var timeoutLoops = timeoutSeconds * 10

		var statSleep = time.Duration(syncWaitSeconds) * time.Second
		for i := 1; i <= timeoutLoops; i++ {
			cmd.out.Verbose("Checking that a file can sync: Attempt #%d", i)
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
