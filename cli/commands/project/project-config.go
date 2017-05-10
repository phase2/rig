package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/phase2/rig/cli/util"
	"gopkg.in/yaml.v2"
)

var configFile string

type ProjectScript struct {
	Alias       string
	Description string
	Run         []string
}

type ProjectConfig struct {
	Scripts   map[string]*ProjectScript
	Namespace string
	Version   string
	Bin       string
}

// Given a project configuration file will load YAML, validate it for purpose,
// and return a normalized object.
func GetProjectConfigFromFile(filename string) ProjectConfig {
	config := LoadYamlFromFile(filename)
	if err := ValidateConfig(config); err != nil {
		util.Logger().Error.Printf("Error in Project Config: %s", filename)
		util.Logger().Error.Fatalf("%s", err)
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
// @todo do this in a more formal way. See docker/libcompose for an example.
func ValidateConfig(config ProjectConfig) error {
	if len(config.Version) == 0 {
		return fmt.Errorf("No 'version' property detected.")
	}

	if config.Version != "1.0" {
		return fmt.Errorf("Version '1.0' is the only supported value, found '%s'.", config.Version)
	}

	return nil
}

// Given a filename, ensures it exists and unmarshals the raw Yaml.
func LoadYamlFromFile(filename string) ProjectConfig {
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		util.Logger().Error.Fatalf("Project configuration file not found at '%s'", filename)
	}

	return LoadYaml(yamlFile)
}

// Set up the output streams (and colors) to stream command output if verbose is configured
func LoadYaml(in []byte) ProjectConfig {
	var config ProjectConfig
	if err := yaml.Unmarshal(in, &config); err != nil {
		util.Logger().Error.Printf("YAML Parsing Error")
		util.Logger().Error.Fatal(err)
	}

	return config
}

// Initialize the configuration system.
func ConfigInit() {
	configFile = os.Getenv("RIG_PROJECT_CONFIG_FILE")
	if len(configFile) == 0 {
		configFile = "./.outrigger.yml"
	}
}

// Get the absolute path to the configuration file.
func GetConfigPath() string {
	filename, _ := filepath.Abs(configFile)
	return filename
}
