package commands

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
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
type P struct {
	Scripts map[string]string
	Version string
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

			var (
				sysShell      = "sh"
				sysCommandArg = "-c"
			)

			shellCmd := exec.Command(sysShell, sysCommandArg, val)
			shellCmd.Dir = filepath.Dir(cmd.GetConfigPath(c))
/*
			shellCmd := exec.Cmd{
				Path: sysShell,
				Args: []string{sysCommandArg, val},
				Dir: filepath.Dir(cmd.GetConfigPath(c)),
			}
*/
			if err := util.StreamCommand(shellCmd); err != nil {
				cmd.out.Error.Fatalf("Error running project script '%s': %s", alias, err)
			}
		} else {
			util.Logger().Error.Printf("Unrecognized script '%s'", alias)
		}
	}

	return nil
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
func (cmd *Project) GetProjectConfig(filename string) P {
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		util.Logger().Error.Fatalf("Project configuration file not found at '%s'", filename)
	}

	var config P
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		util.Logger().Error.Printf("Error loading YAML project config file: '%s'", filename)
		util.Logger().Error.Fatal(err)
	}

	if len(config.Version) == 0 {
		util.Logger().Error.Printf("No 'version' property detected for your configuration file '%s'", filename)
	}

	if config.Version != "1.0" {
		util.Logger().Error.Printf("Version '1.0' is the only supported value in '%s'", filename)
	}

	return config
}
