package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	containertypes 	"github.com/docker/docker/api/types/container"
	volumetypes 		"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
  "github.com/phase2/rig/cli/commands/project"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
  "gopkg.in/yaml.v2"
	"time"
	"net"
	"github.com/docker/docker/daemon/exec"
)

type Project struct {
	BaseCommand
}

func (cmd *Project) Commands() cli.Command {
	project.ConfigInit()
	command := cli.Command{
		Name:        "project",
		Usage:       "Run a project script from configuration.",
		Description: "Configure scripts representing core operations of the project in a Rig configuration file.\n\n\tThis Yaml file by default is ./.outrigger.yml. It can be overridden by setting an environment variable $RIG_PROJECT_CONFIG_FILE.",
		Category:    "Development",
		Before:      cmd.Before,
		Subcommands: cmd.GetScriptsAsSubcommands(project.GetConfigPath()),
	}

	syncStart := cli.Command{
		Name:        "sync:start",
		Usage:       "Start a unison sync on local project directory. Optionally provide a volume name. Volume name will be discovered in the following order: argument, outrigger project config, docker-compose file, current directory name",
		ArgsUsage:   "[optional volume name]",
		Before:      cmd.Before,
		Action:      cmd.RunSyncStart,
	}

	command.Subcommands = append(command.Subcommands, syncStart)

	return command
}

// Processes script configuration into formal subcommands.
func (cmd *Project) GetScriptsAsSubcommands(filename string) []cli.Command {
	var scripts = cmd.GetProjectScripts(filename)

	var commands = []cli.Command{}
	for id, script := range scripts {
		if len(script.Run) > 0 {
			command := cli.Command{
				Name:        id,
				Usage:       script.Description,
				Description: fmt.Sprintf("%s\n\n\tThis command was configured in %s\n\n\tThere are %d steps in this script and any 'extra' arguments will be appended to the final step.", script.Description, filename, len(script.Run)),
				ArgsUsage:   "<args passed to last step>",
				Before:      cmd.Before,
				Action:      cmd.Run,
			}

			if len(script.Alias) > 0 {
				command.Aliases = []string{script.Alias}
			}

			commands = append(commands, command)
		}
	}

	return commands
}

// Return the help for all the scripts.
func (cmd *Project) Run(c *cli.Context) error {
	var scripts = cmd.GetProjectScripts(project.GetConfigPath())

	key := c.Command.Name
	if script, ok := scripts[key]; ok {
		cmd.out.Verbose.Printf("Executing '%s' for '%s'", key, script.Description)
		cmd.addCommandPath(project.GetConfigPath())
		dir := filepath.Dir(project.GetConfigPath())
		for step, val := range script.Run {
			cmd.out.Verbose.Printf("Step %d: Executing '%s' as '%s'", step+1, key, val)
			// If this is the last step, append any further args to the end of the command.
			if len(script.Run) == step+1 {
				val = val + " " + strings.Join(c.Args(), " ")
			}
			shellCmd := cmd.GetCommand(val)
			shellCmd.Dir = dir

			if _, stderr, exitCode := util.PassthruCommand(shellCmd); exitCode != 0 {
				cmd.out.Error.Printf("Error running project script '%s' on step %d: %s", key, step+1, stderr)
				os.Exit(exitCode)
			}
		}
	} else {
		util.Logger().Error.Printf("Unrecognized script '%s'", key)
	}

	return nil
}

// Construct a command to execute a configured script.
// @see https://github.com/medhoover/gom/blob/staging/config/command.go
func (cmd *Project) GetCommand(val string) *exec.Cmd {
	var (
		sysShell      = "sh"
		sysCommandArg = "-c"
	)
	if runtime.GOOS == "windows" {
		sysShell = "cmd"
		sysCommandArg = "/c"
	}

	return exec.Command(sysShell, sysCommandArg, val)
}

// Load the scripts from the project-specific configuration.
func (cmd *Project) GetProjectScripts(filename string) map[string]*project.ProjectScript {
	scripts := project.GetProjectConfigFromFile(filename).Scripts
	// We can hard-wire scripts here by assigning: scripts["name"] = &project.ProjectScript{}

	return scripts
}

// Override the PATH environment variable for further shell executions.
// This is used on POSIX systems for lookup of scripts.
func (cmd *Project) addCommandPath(filename string) error {
	binDir := project.GetProjectConfigFromFile(filename).Bin
	cmd.out.Verbose.Printf("Adding '%s' to the PATH for script execution.", binDir)
	path := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", path, binDir))

	return nil
}

/////////////////////////////////////////////////////////////////////////
// Sync Commands
/////////////////////////////////////////////////////////////////////////
const UNISON_PORT = 5000

// Start the unison sync process
func (cmd *Project) RunSyncStart(ctx *cli.Context) error {
  project.ConfigInit();
  config := project.GetProjectConfigFromFile(project.GetConfigPath())
  volumeName := cmd.GetVolumeName(ctx, config)

	client, err := client.NewEnvClient()
	if err != nil {
		cmd.out.Error.Fatal("Unable to create Docker Client")
	}

	cmd.out.Info.Printf("Starting sync volume: %s", volumeName)
	client.VolumeCreate(context.Background(), volumetypes.VolumesCreateBody{ Name: volumeName })


	cmd.out.Info.Println("Starting unison container")
	hostConfig := &containertypes.HostConfig{
		AutoRemove: true,
		Binds: []string{fmt.Sprintf("%s:/unison", volumeName)},
	}
	containerConfig := &containertypes.Config{
		Env: []string{"UNISON_DIR=/unison"},
		Labels: map[string]string{
			"com.dnsdock.name": volumeName,
			"com.dnsdock.image": "volume.outrigger",
		},
		Image: "outrigger/unison:latest",
	}
	container, err := client.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, volumeName)
	if err != nil {
		cmd.out.Error.Fatalf("Error starting sync container %s: %v", volumeName, err)
	}

	cmd.WaitForUnisonContainer(client, container.ID)

	cmd.out.Info.Println("Initializing sync")

	// Remove the log file, the existence of the log file will mean that sync is up and running
	//exec.Command("rm", "-f", fmt.Sprintf("%s.log", volumeName)).Run()


  cmd.out.Info.Printf("Volume name: %s", volumeName)
	return nil
}

// Find the volume name through a variety of fall backs
func (cmd *Project) GetVolumeName(ctx *cli.Context, config project.ProjectConfig) string {
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
      cmd.out.Verbose.Printf("Volume: Name: %s External: %t", name, volume.External);
      if strings.HasSuffix(name, "-sync") && volume.External{
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

type ComposeFile struct {
  Volumes map[string]Volume
}

type Volume struct {
  External bool
}

// Load the proper compose file
func (cmd *Project) LoadComposeFile() (*ComposeFile, error) {
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

func (cmd *Project) WaitForUnisonContainer(client *client.Client, containerId string) {
	cmd.out.Info.Println("Waiting for container to start")
	containerData, err := client.ContainerInspect(context.Background(), containerId)
	if err != nil {
		cmd.out.Error.Fatalf("Error inspecting sync container %s: %v", containerId, err)
	}
	for i := 1; i <= 100; i++ {
		//if err := exec.Command("nc", "-z", containerData.NetworkSettings.IPAddress, UNISON_PORT).Run(); err != nil {
		if _, err := net.Dial("tcp", fmt.Sprintf("%s:%s", containerData.NetworkSettings.IPAddress, UNISON_PORT)); err == nil {
			return
		} else {
			time.Sleep(time.Duration(100) * time.Millisecond)
		}
	}
	cmd.out.Error.Fatal("Sync container failed to start!")
}

