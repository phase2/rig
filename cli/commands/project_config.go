package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/phase2/rig/cli/util"
	"gopkg.in/yaml.v2"
)

type ProjectScript struct {
	Alias       string
	Description string
	Run         []string
}

type Sync struct {
	Volume    string
	Ignore 		[]string
}

type ProjectConfig struct {
	File string
	Path string

	Scripts   map[string]*ProjectScript
	Sync   		*Sync
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

	if err := ValidateProjectConfig(config); err != nil {
		logger.Error.Fatalf("Error in %s: %s", filename, err)
	}

	if len(config.Bin) == 0 {
		config.Bin = "./bin"
	}

	for id, script := range config.Scripts {
		if len(script.Description) == 0 {
			config.Scripts[id].Description = fmt.Sprintf("Configured operation for '%s'", id)
		}
	}

	return config
}


// Ensures our configuration data structure conforms to our ad hoc schema.
// @TODO do this in a more formal way. See docker/libcompose for an example.
func ValidateProjectConfig(config *ProjectConfig) error {
	if len(config.Version) == 0 {
		return fmt.Errorf("No 'version' property detected.")
	}

	if config.Version != "1.0" {
		return fmt.Errorf("Version '1.0' is the only supported value, found '%s'.", config.Version)
	}

	if config.Scripts != nil {
		for id, script := range config.Scripts {
			if len(script.Run) > 10 {
				util.Logger().Warning.Printf("Project script '%s' has more than 10 run items (%d). You should create a shell script to contain those.", id, len(script.Run))
			}
		}
	}

	return nil
}
