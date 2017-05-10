package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/phase2/rig/cli/commands/project"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Project struct {
	BaseCommand
}

func (cmd *Project) Commands() cli.Command {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:   "config",
			Value:  "./.outrigger.yml",
			Usage:  "Path to the project-specific configuration for Outrigger.",
			EnvVar: "RIG_PROJECT_CONFIG_FILE",
		},
	}

	command := cli.Command{
		Name:      "project",
		Usage:     "Run a configured project script",
		ArgsUsage: "<script to run>",
		Category:  "Development",
		Flags:     flags,
		Before:    cmd.Before,
		Action:    cmd.Run,
	}

	return command
}

// Return the help for all the scripts.
func (cmd *Project) Run(c *cli.Context) error {
	var scripts = cmd.GetProjectScripts(cmd.GetConfigPath(c))

	key := c.Args().Get(0)
	if len(key) == 0 {
		cmd.out.Info.Print("These are the local project scripts:")
		for k, v := range scripts {
			cmd.out.Info.Printf("\t - %s [%s]:\t\t\t%s", k, v.Alias, v.Description)
		}
	} else {
		if script, ok := scripts[key]; ok {
			cmd.out.Verbose.Printf("Executing '%s' as '%s'", key, script.Description)
			dir := filepath.Dir(cmd.GetConfigPath(c))
			for step, val := range script.Run {
				cmd.out.Verbose.Printf("Executing '%s' as '%s'", key, val)
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

	scripts["scripts"] = &project.ProjectScript{
		Alias:       "h",
		Description: "Information about the available project scripts.",
		Run:         []string{"rig project"},
	}

	return scripts
}

// Get the absolute path to the configuration file.
func (cmd *Project) GetConfigPath(c *cli.Context) string {
	filename, _ := filepath.Abs(c.String("config"))
	return filename
}
