package commands

import (
	"fmt"
  "io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

  "github.com/phase2/rig/cli/commands/project"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
  "gopkg.in/yaml.v2"
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
		Usage:       "Start a unison sync on local project directory",
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

// Start the unison sync process
func (cmd *Project) RunSyncStart(c *cli.Context) error {
  project.ConfigInit();
  //config := project.GetProjectConfigFromFile(project.GetConfigPath())

  volume := cmd.GetVolumeName()
  cmd.out.Info.Printf("Volume name: %s", volume)
	return nil
}

func (cmd *Project) GetVolumeName() string {
  // 1. Check for argument
  // 2. Check for config
  // 3. Parse compose file looking for an external volume named *-sync
  if composeConfig, err := cmd.LoadComposeFile(); err == nil {
    for name, volume := range composeConfig.Volumes {
      cmd.out.Info.Printf("Name: %s", name);
      cmd.out.Info.Printf("External: %t", volume.External);
      if strings.HasSuffix(name, "-sync") && volume.External{
        return name
      }
    }
  }

  // 4. Use local dir for the volume name
  cmd.out.Error.Fatal("Use local dir name for volume")
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

