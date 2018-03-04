package commands

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
)

// Project is the command enabling projects to define their own custom commands in a project based configuration file
type Project struct {
	BaseCommand
	Config *ProjectConfig
}

// Commands returns the operations supported by this command
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

// GetScriptsAsSubcommands Processes script configuration into formal subcommands.
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

// Run executes the specified `rig project` script
func (cmd *Project) Run(c *cli.Context) error {
	cmd.out.Verbose("Loaded project configuration from %s", cmd.Config.Path)
	if cmd.Config.Scripts == nil {
		cmd.out.Channel.Error.Fatal("There are no scripts discovered in: %s", cmd.Config.File)
	}

	key := strings.TrimPrefix(c.Command.Name, "run:")
	if script, ok := cmd.Config.Scripts[key]; !ok {
		return cmd.Failure(fmt.Sprintf("Unrecognized script '%s'", key), "SCRIPT-NOT-FOUND", 12)
	} else {
		eval := ProjectEval{cmd.out, cmd.Config}
		if exitCode := eval.ProjectScriptRun(script, c.Args()); exitCode != 0 {
			return cmd.Failure(fmt.Sprintf("Failure running project script '%s'", key), "COMMAND-ERROR", exitCode)
		}
	}

	return cmd.Success("")
}

// ScriptRunHelp generates help details based on script configuration.
func (cmd *Project) ScriptRunHelp(script *ProjectScript) string {
	help := fmt.Sprintf("\n\nSCRIPT STEPS:\n\t- ")
	help = help + strings.Join(script.Run, "\n\t- ") + " [args...]\n"

	return help
}
