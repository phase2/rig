package commands

import (
	"io/ioutil"
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
scripts:
  build: "some build command && another build command"
  start: "bin/start.sh"
  stop:  "docker-compose stop"
  clean: "./cleanup.sh"
```
*/
type P struct {
	Scripts map[string]string
}

func (cmd *Project) Commands() cli.Command {
	command := cli.Command{
		Name:   "project",
		Usage:  "Project related commands",
		Before: cmd.Before,
		Action: cmd.Run,
	}

	// Add subcommands

	return command

}

func (cmd *Project) Run(c *cli.Context) error {
	var scripts = cmd.GetLocalScripts()

	cmd.out.Info.Print("These are the local project scripts:")
	for k := range scripts {
		cmd.out.Info.Printf("\t - %s", k)
	}
	return nil
}

// Load a .rig.yml file from the current directory
func (cmd *Project) GetLocalScripts() map[string]string {
	filename, _ := filepath.Abs("./.rig.yml")
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		util.Logger().Error.Fatalf("Project configuration file not found at '%s'", filename)
	}

	var config P
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		util.Logger().Error.Printf("Error loading YAML project config file: '%s'", filename)
		util.Logger().Error.Fatal(err)
	}

	return config.Scripts
}
