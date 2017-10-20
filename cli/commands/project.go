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
	Config *ProjectConfig
}

func (cmd *Project) Commands() []cli.Command {
	cmd.Config = NewProjectConfig()

	command := cli.Command{
		Name:        "project",
		Usage:       "Run project-specific commands.",
		Description: "Run project-specific commands as part of development.\n\n\tConfigured scripts are driven by an Outrigger configuration file expected at your project root directory.\n\n\tBy default, this is a YAML file named 'outrigger.yml' with fallback to '.outrigger.yml'. It can be overridden by setting an environment variable $RIG_PROJECT_CONFIG_FILE.",
		Aliases:     []string{"run"},
		Category:    "Development",
		Before:      cmd.Before,
	}

	create := ProjectCreate{}
	command.Subcommands = append(command.Subcommands, create.Commands()...)

	sync := ProjectSync{}
	command.Subcommands = append(command.Subcommands, sync.Commands()...)

	doctor := ProjectDoctor{}
	command.Subcommands = append(command.Subcommands, doctor.Commands()...)

	if subcommands := cmd.GetScriptsAsSubcommands(command.Subcommands); subcommands != nil {
		command.Subcommands = append(command.Subcommands, subcommands...)
	}

	return []cli.Command{command}
}

// Processes script configuration into formal subcommands.
func (cmd *Project) GetScriptsAsSubcommands(otherSubcommands []cli.Command) []cli.Command {
	cmd.Config.ValidateProjectScripts(otherSubcommands)

	if cmd.Config.Scripts == nil {
		return nil
	}

	var commands = []cli.Command{}
	for id, script := range cmd.Config.Scripts {
		if len(script.Run) > 0 {
			command := cli.Command{
				Name:        fmt.Sprintf("run:%s", id),
				Usage:       script.Description,
				Description: fmt.Sprintf("%s\n\n\tThis command was configured in %s\n\n\tThere are %d steps in this script and any 'extra' arguments will be appended to the final step.", script.Description, cmd.Config.File, len(script.Run)),
				ArgsUsage:   "<args passed to last step>",
				Category:    "Configured Scripts",
				Before:      cmd.Before,
				Action:      cmd.Run,
			}

			if len(script.Alias) > 0 {
				command.Aliases = []string{script.Alias}
			}
			command.Description = command.Description + cmd.ScriptRunHelp(script)

			commands = append(commands, command)
		}
	}

	return commands
}

// Execute the selected projec script.
func (cmd *Project) Run(c *cli.Context) error {
	cmd.out.Verbose.Printf("Loaded project configuration from %s", cmd.Config.Path)
	if cmd.Config.Scripts == nil {
		cmd.out.Error.Fatal("There are no scripts discovered in: %s", cmd.Config.File)
	}

	key := strings.TrimPrefix(c.Command.Name, "run:")
	if script, ok := cmd.Config.Scripts[key]; ok {
		cmd.out.Verbose.Printf("Executing '%s': %s", key, script.Description)
		cmd.AddCommandPath()
		dir := filepath.Dir(cmd.Config.Path)

		shellCmd := cmd.GetCommand(script.Run, c.Args(), dir)

		cmd.out.Verbose.Printf("Executing '%s' as '%s'", key, shellCmd.Path)
		if exitCode := util.PassthruCommand(shellCmd, true); exitCode != 0 {
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
func (cmd *Project) GetCommand(steps []string, extra []string, workingDirectory string) *exec.Cmd {
	// Concat the commands together adding the args to this command as args to the last step
	scriptCommands := strings.Join(steps, cmd.GetCommandSeparator()) + " " + strings.Join(extra, " ")

	var command *exec.Cmd
	if runtime.GOOS == "windows" {
		command = exec.Command("cmd", "/c", scriptCommands)
	}

	command = exec.Command("sh", "-c", scriptCommands)
	command.Dir = workingDirectory

	return command
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
func (cmd *Project) AddCommandPath() {
	binDir := cmd.Config.Bin
	if binDir != "" {
		cmd.out.Verbose.Printf("Adding '%s' to the PATH for script execution.", binDir)
		path := os.Getenv("PATH")
		os.Setenv("PATH", fmt.Sprintf("%s%c%s", binDir, os.PathListSeparator, path))
	}
}

// Generate help details based on script configuration.
func (cmd *Project) ScriptRunHelp(script *ProjectScript) string {
	help := fmt.Sprintf("\n\nSCRIPT STEPS:\n\t- ")
	help = help + strings.Join(script.Run, "\n\t- ") + " [args...]\n"

	return help
}
