package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phase2/rig/cli/commands/project"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Project struct {
	BaseCommand
}

func (cmd *Project) Commands() []cli.Command {
	project.ConfigInit()
	command := cli.Command{
		Name:        "project",
		Usage:       "Run project-specific commands.",
		Description: "Run project-specific commands as part of development.\n\n\tConfigured scripts are driven by an Outrigger configuration file expected at your project root directory.\n\n\tBy default, this is a YAML file named '.outrigger.yml'. It can be overridden by setting an environment variable $RIG_PROJECT_CONFIG_FILE.",
		Aliases:     []string{"run"},
		Category:    "Development",
		Before:      cmd.Before,
		Subcommands: cmd.GetScriptsAsSubcommands(project.GetConfigPath()),
	}

	sync := ProjectSync{}
	command.Subcommands = append(command.Subcommands, sync.Commands()...)

	return []cli.Command{command}
}

// Processes script configuration into formal subcommands.
func (cmd *Project) GetScriptsAsSubcommands(filename string) []cli.Command {
	scripts, err := cmd.GetProjectScripts(filename)
	if err != nil {
		cmd.out.Verbose.Printf("%s", err)
		return []cli.Command{}
	}

	var commands = []cli.Command{}
	for id, script := range scripts {
		if len(script.Run) > 0 {
			command := cli.Command{
				Name:        fmt.Sprintf("run:%s", id),
				Usage:       script.Description,
				Description: fmt.Sprintf("%s\n\n\tThis command was configured in %s\n\n\tThere are %d steps in this script and any 'extra' arguments will be appended to the final step.", script.Description, filename, len(script.Run)),
				ArgsUsage:   "<args passed to last step>",
				Category:    "Configured Scripts",
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
	scripts, err := cmd.GetProjectScripts(project.GetConfigPath())
	if err != nil {
		cmd.out.Error.Fatal(err)
	}

	key := strings.TrimPrefix(c.Command.Name, "run:")
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
func (cmd *Project) GetProjectScripts(filename string) (map[string]*project.ProjectScript, error) {
	config, err := project.GetProjectConfigFromFile(filename)
	// We can hard-wire scripts here by assigning: scripts["name"] = &project.ProjectScript{}
	// If we do it here, they will be set up as "configured" subcommands.

	return config.Scripts, err
}

// Override the PATH environment variable for further shell executions.
// This is used on POSIX systems for lookup of scripts.
func (cmd *Project) addCommandPath(filename string) error {
	config, err := project.GetProjectConfigFromFile(filename)
	if err != nil {
		cmd.out.Error.Fatal(err)
	}
	binDir := config.Bin
	cmd.out.Verbose.Printf("Adding '%s' to the PATH for script execution.", binDir)
	path := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s:%s", path, binDir))

	return nil
}
