package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Project struct {
	BaseCommand
}

func (cmd *Project) Commands() []cli.Command {
	command := cli.Command{
		Name:        "project",
		Usage:       "Run project-specific commands.",
		Description: "Run project-specific commands as part of development.\n\n\tConfigured scripts are driven by an Outrigger configuration file expected at your project root directory.\n\n\tBy default, this is a YAML file named '.outrigger.yml'. It can be overridden by setting an environment variable $RIG_PROJECT_CONFIG_FILE.",
		Aliases:     []string{"run"},
		Category:    "Development",
		Before:      cmd.Before,
	}

	if subcommands := cmd.GetScriptsAsSubcommands(); subcommands != nil {
		command.Subcommands = subcommands
	}

	sync := ProjectSync{}
	command.Subcommands = append(command.Subcommands, sync.Commands()...)

	return []cli.Command{command}
}

// Processes script configuration into formal subcommands.
func (cmd *Project) GetScriptsAsSubcommands() []cli.Command {
	config := NewProjectConfig()

	if config.Scripts == nil {
		return nil
	}

	var commands = []cli.Command{}
	for id, script := range config.Scripts {
		if len(script.Run) > 0 {
			command := cli.Command{
				Name:        fmt.Sprintf("run:%s", id),
				Usage:       script.Description,
				Description: fmt.Sprintf("%s\n\n\tThis command was configured in %s\n\n\tThere are %d steps in this script and any 'extra' arguments will be appended to the final step.", script.Description, config.File, len(script.Run)),
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
	config := NewProjectConfig()

	if config.Scripts == nil {
		cmd.out.Error.Fatal("There are no scripts discovered in: %s", config.File)
	}

	key := strings.TrimPrefix(c.Command.Name, "run:")
	if script, ok := config.Scripts[key]; ok {
		cmd.out.Verbose.Printf("Executing '%s': %s", key, script.Description)
		cmd.addCommandPath(config)
		dir := filepath.Dir(config.Path)

		// Concat the commands together adding the args to this command as args to the last step
		scriptCommands := strings.Join(script.Run, cmd.GetCommandSeparator()) + " " + strings.Join(c.Args(), " ")

		shellCmd := cmd.GetCommand(scriptCommands)
		shellCmd.Dir = dir

		cmd.out.Verbose.Printf("Executing '%s' as '%s'", key, scriptCommands)
		if exitCode := util.PassthruCommand(shellCmd); exitCode != 0 {
			cmd.out.Error.Printf("Error running project script '%s': %d", key, exitCode)
			os.Exit(exitCode)
		}
	} else {
		cmd.out.Error.Printf("Unrecognized script '%s'", key)
	}

	return nil
}

// Construct a command to execute a configured script.
// @see https://github.com/medhoover/gom/blob/staging/config/command.go
func (cmd *Project) GetCommand(val string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", val)
	}

	return exec.Command("sh", "-c", val)
}

// Get command separator based on platform.
func (cmd *Project) GetCommandSeparator() string {
	if runtime.GOOS == "windows" {
		return " & "
	}

	return " && "
}

// Override the PATH environment variable for further shell executions.
// This is used on POSIX systems for lookup of scripts.
func (cmd *Project) addCommandPath(config *ProjectConfig) {
	if config.Bin != "" {
		binDir := config.Bin
		cmd.out.Verbose.Printf("Adding '%s' to the PATH for script execution.", binDir)
		path := os.Getenv("PATH")
		os.Setenv("PATH", fmt.Sprintf("%s:%s", path, binDir))
	}
}
