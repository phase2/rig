package util

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type ProjectConfig struct {
	Scripts map[string]string
	Version string
}

func LoadYamlFromFile(filename string) ProjectConfig {
  yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		Logger().Error.Fatalf("Project configuration file not found at '%s'", filename)
	}

  return LoadYaml(yamlFile)
}

// Set up the output streams (and colors) to stream command output if verbose is configured
func LoadYaml(in []byte) ProjectConfig {
	var config ProjectConfig
	if err := yaml.Unmarshal(in, &config); err != nil {
		Logger().Error.Printf("YAML Parsing Error")
		Logger().Error.Fatal(err)
	}

  return config
}
