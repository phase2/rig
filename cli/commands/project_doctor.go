package commands

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"runtime"

	"github.com/phase2/rig/cli/util"
)

type ProjectDoctor struct {
	BaseCommand
	Config *ProjectConfig
}

type ConditionSeverity int

const (
	ConditionSeverityINFO    ConditionSeverity = 0
	ConditionSeverityWARNING ConditionSeverity = 1
	ConditionSeverityERROR   ConditionSeverity = 2
)

type Condition struct {
	ID           string
	Name         string
	Test         []string
	Diagnosis    string
	Healthy      string
	Prescription string
	Severity     ConditionSeverity
}

func (cmd *ProjectDoctor) Commands() []cli.Command {
	cmd.Config = NewProjectConfig()

	diagnose := cli.Command{
		Name:        "doctor:diagnose",
		Aliases:     []string{"doctor"},
		Usage:       "Run to evaluate project-level environment problems.",
		Description: "This command validates known problems with the project environment. The rules can be extended via the 'doctor' section of the project configuration.",
		Before:      cmd.Before,
		Action:      cmd.RunAnalysis,
	}

	compendium := cli.Command{
		Name:        "doctor:conditions",
		Usage:       "Learn all the rules applied by the doctor:diagnose command.",
		Description: "Display all the conditions for which the doctor:diagnose command will check.",
		Before:      cmd.Before,
		Action:      cmd.RunCompendium,
	}

	return []cli.Command{diagnose, compendium}
}

// Run the diagnosis process.
func (cmd *ProjectDoctor) RunAnalysis(ctx *cli.Context) error {
	compendium, _ := cmd.GetConditionList()
	if err := cmd.AnalyzeConditionList(compendium); err != nil {
		return cli.NewExitError(fmt.Sprintf("%v", err), 1)
	}

	return nil
}

// List all conditions to be checked in the analysis.
func (cmd *ProjectDoctor) RunCompendium(ctx *cli.Context) error {
	compendium, _ := cmd.GetConditionList()
	cmd.PrintConditionList(compendium)

	return nil
}

// Check whether any of the available conditions is met.
func (cmd *ProjectDoctor) AnalyzeConditionList(conditions []Condition) error {
	var returnVal error
	for _, condition := range conditions {
		err := cmd.Analyze(condition)
		if err == nil {
			cmd.out.Info.Printf("%s (%s [%s])", condition.Healthy, condition.Name, condition.ID)
		} else if err != nil {
			switch condition.Severity {
			case ConditionSeverityWARNING:
				cmd.out.Warning.Printf("%s", condition.ToString())
			case ConditionSeverityERROR:
				cmd.out.Error.Printf("%s", condition.ToString())
				if returnVal == nil {
					returnVal = errors.New("Diagnosis found a failing condition.")
				}
			default:
				cmd.out.Info.Printf("%s", condition.ToString())
			}
		}
	}
  fmt.Println("")

	return returnVal
}

// Assemble a list of all conditions.
func (cmd *ProjectDoctor) GetConditionList() ([]Condition, error) {
	var conditions = []Condition{}

	// @TODO it would be nice to make this an error and only include it if the
	// project is definitely using Unison.
	if runtime.GOOS != "linux" {
		conditionSyncContainerExists := Condition{
			ID:           "sync-container-missing",
			Name:         "Sync Container Missing",
			Test:         []string{"exit 1"},
			Diagnosis:    "The Sync container for this project is not available.",
			Prescription: "Run 'rig project sync:start' before beginning work. This command may be included in other project-specific tasks.",
			Healthy:      "Sync container present to facilitate Unison filesystem support.",
			Severity:     ConditionSeverityWARNING,
		}
		conditions = append(conditions, conditionSyncContainerExists)
	}

	return conditions, nil
}

// Print a list of all conditions.
func (cmd *ProjectDoctor) PrintConditionList(conditions []Condition) {
	for _, condition := range conditions {
		fmt.Printf("---\n%s\n---\n", condition.ToString())
	}
}

// Evaluate if the condition criteria is met.
func (cmd *ProjectDoctor) Analyze(c Condition) error {
	// @todo move some of the command-wrangling to utility methods.
	project := Project{BaseCommand{machine: cmd.machine, out: cmd.out}, cmd.Config}
	project.AddCommandPath()
	shellCmd := project.GetCommand(c.Test, []string{}, ".")

	if exitCode := util.PassthruCommand(shellCmd, false); exitCode != 0 {
		cmd.out.Verbose.Printf("Condition analysis failed with code %d", exitCode)
		return errors.New("Analysis confirmed for condition.")
	}

	return nil
}

// Convert the condition to a readable entry.
func (c Condition) ToString() string {
	return fmt.Sprintf("%s (%s)\n\tDESCRIPTION: %s\n\tSOLUTION: %s\n\t[%s]", c.Name, c.SeverityLabel(), c.Diagnosis, c.Prescription, c.ID)
}

// Convert the severity code to a human-readable label.
func (c Condition) SeverityLabel() string {
	var label string
	switch c.Severity {
	case ConditionSeverityERROR:
		label = "ERROR"
		break
	case ConditionSeverityWARNING:
		label = "WARNING"
		break
	case ConditionSeverityINFO:
	default:
		label = "INFO"
		break
	}

	return label
}
