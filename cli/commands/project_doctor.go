package commands

import (

  	"github.com/urfave/cli"
)

type ProjectDoctor struct {
	BaseCommand
}

type ConditionSeverity int

var (
	ConditionSeverityINFO ConditionSeverity = 0
	ConditionSeverityWARNING ConditionSeverity = 1
	ConditionSeverityERROR ConditionSeverity = 2
)

type Condition struct {
  Name string
  Diagnosis string
  Prescription string
  Severity ConditionSeverity
}

func (cmd *ProjectDoctor) Commands() []cli.Command {
	doctor := cli.Command{
		Name:        "doctor",
		Aliases:     []string{"diagnose"},
		Usage:       "Run to evaluate project-level environment problems.",
		Description: "This command validates known problems with the project environment. It only operates if a project configuration file is active. The rules can be extended via the 'doctor' section of the project configuration.",
		Before:      cmd.Before,
		Action:      cmd.Run,
	}

	return []cli.Command{doctor}
}

// Run the diagnosis process.
func (cmd *ProjectDoctor) Run(ctx *cli.Context) error {
  //config := cmd.getConditionSet();
  return nil

}

func (cmd *ProjectDoctor) getConditionSet() error {
  //NewProjectConfig()
  return nil
}
