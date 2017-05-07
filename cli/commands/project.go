package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type Project struct {
	BaseCommand
}

/*
Here is a sample config file

```
version: 1.0
namespace: project-name
scripts:
  build: "some build command && another build command"
  start: "bin/start.sh"
  stop:  "docker-compose stop"
  clean: "./cleanup.sh"
```
*/

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
		Name: "project",
		Usage: "Project related commands",
		Category: "Development",
		Subcommands: []cli.Command{},
		Before: cmd.Before,
	}

	command.Subcommands = append(command.Subcommands, cli.Command{
		Name: "run",
		Usage: "Run a configured script",
		ArgsUsage: "<script to run>",
		Flags: flags,
		Before: cmd.Before,
		Action: cmd.Run,
	});

	return command
}

func (cmd *Project) Run(c *cli.Context) error {
	var scripts = cmd.GetProjectScripts(cmd.GetConfigPath(c))

	alias := c.Args().Get(0);
	if len(alias) == 0 {
		cmd.out.Info.Print("These are the local project scripts:")
		for k := range scripts {
			cmd.out.Info.Printf("\t - %s", k)
		}
	}	else {
		if val, ok := scripts[alias]; ok {
		  cmd.out.Verbose.Printf("Executing '%s' as '%s'", alias, val)
			shellCmd := cmd.GetCommand(val)
			shellCmd.Dir = filepath.Dir(cmd.GetConfigPath(c))

			if _, stderr, exitCode := util.PassthruCommand(shellCmd); exitCode != 0 {
				cmd.out.Error.Fatalf("Error running project script '%s': %s", alias, stderr)
				os.Exit(exitCode)
			}
		} else {
			util.Logger().Error.Printf("Unrecognized script '%s'", alias)
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
func (cmd *Project) GetProjectScripts(filename string) map[string]string {
	return cmd.GetProjectConfig(filename).Scripts;
}

func (cmd *Project) GetConfigPath(c *cli.Context) string {
	filename, _ := filepath.Abs(c.String("config"))
	return filename
}

// Load a project-specific rig configuration file from the current directory
func (cmd *Project) GetProjectConfig(filename string) util.ProjectConfig {
	var config util.ProjectConfig
	config = util.LoadYamlFromFile(filename)

	if len(config.Version) == 0 {
		util.Logger().Error.Printf("No 'version' property detected for your configuration file '%s'", filename)
	}

	if config.Version != "1.0" {
		util.Logger().Error.Printf("Version '1.0' is the only supported value in '%s'", filename)
	}

	return config
}
