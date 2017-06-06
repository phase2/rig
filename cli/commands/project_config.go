package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

type ProjectScript struct {
	Alias       string
	Description string
	Run         []string
}

type Sync struct {
	Volume string
	Ignore []string
}

type ProjectConfig struct {
	File string
	Path string

	Scripts   map[string]*ProjectScript
	Sync      *Sync
	Namespace string
	Version   string
	Bin       string
}

// Create a new ProjectConfig using configured or default locations
func NewProjectConfig() *ProjectConfig {
	projectConfigFile := os.Getenv("RIG_PROJECT_CONFIG_FILE")
	if projectConfigFile == "" {
		projectConfigFile = "./.outrigger.yml"
	}
	return NewProjectConfigFromFile(projectConfigFile)
}

// Create a new ProjectConfig from the specified file
func NewProjectConfigFromFile(filename string) *ProjectConfig {
	logger := util.Logger()

	filepath, _ := filepath.Abs(filename)
	config := &ProjectConfig{
		File: filename,
		Path: filepath,
	}

	yamlFile, err := ioutil.ReadFile(config.File)
	if err != nil {
		logger.Verbose.Printf("No project configuration file found at: %s", config.File)
		return config
	}

	if err := yaml.Unmarshal(yamlFile, config); err != nil {
		logger.Error.Fatalf("Error parsing YAML config: %v", err)
	}

	if err := config.ValidateConfigVersion(); err != nil {
		logger.Error.Fatalf("Error in %s: %s", filename, err)
	}

	if len(config.Bin) == 0 {
		config.Bin = "./bin"
	}

	for id, script := range config.Scripts {
		if script != nil && script.Description == "" {
			config.Scripts[id].Description = fmt.Sprintf("Configured operation for '%s'", id)
		}
	}

	return config
}

// Ensures our configuration data structure conforms to our ad hoc schema.
// @TODO do this in a more formal way. See docker/libcompose for an example.
func (c *ProjectConfig) ValidateConfigVersion() error {
	if len(c.Version) == 0 {
		return fmt.Errorf("No 'version' property detected.")
	}

	if c.Version != "1.0" {
		return fmt.Errorf("Version '1.0' is the only supported value, found '%s'.", c.Version)
	}

	return nil
}

// Validate the config scripts against a set of rules/norms
func (c *ProjectConfig) ValidateProjectScripts(subcommands []cli.Command) {
	logger := util.Logger()

	if c.Scripts != nil {
		for id, script := range c.Scripts {

			// Check for an empty script
			if script == nil {
				logger.Error.Fatalf("Project script '%s' has no configuration", id)
			}

			// Check for scripts with conflicting aliases with existing subcommands or subcommand aliases
			for _, subcommand := range subcommands {
				if id == subcommand.Name {
					logger.Error.Fatalf("Project script name '%s' conflicts with command name '%s'. Please choose a different script name", id, subcommand.Name)
				} else if script.Alias == subcommand.Name {
					logger.Error.Fatalf("Project script alias '%s' on script '%s' conflicts with command name '%s'. Please choose a different script alias", script.Alias, id, subcommand.Name)
				} else if subcommand.Aliases != nil {
					for _, alias := range subcommand.Aliases {
						if id == alias {
							logger.Error.Fatalf("Project script name '%s' conflicts with command alias '%s' on command '%s'. Please choose a different script name", id, alias, subcommand.Name)
						} else if script.Alias == alias {
							logger.Error.Fatalf("Project script alias '%s' on script '%s' conflicts with command alias '%s' on command '%s'. Please choose a different script alias", script.Alias, id, alias, subcommand.Name)
						}
					}
				}
			}

			// Check for scripts with no run commands
			if script.Run == nil || len(script.Run) == 0 {
				logger.Error.Fatalf("Project script '%s' does not have any run commands.", id)
			} else if len(script.Run) > 10 {
				// Check for scripts with more than 10 run commands
				logger.Warning.Printf("Project script '%s' has more than 10 run items (%d). You should create a shell script to contain those.", id, len(script.Run))
			}
		}
	}

}
