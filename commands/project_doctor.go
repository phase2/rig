package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

type ProjectDoctor struct {
	BaseCommand
	Config *ProjectConfig
}

const (
	ConditionSeverityINFO    string = "info"
	ConditionSeverityWARNING string = "warning"
	ConditionSeverityERROR   string = "error"
)

type Condition struct {
	Id           string
	Name         string
	Test         []string
	Diagnosis    string
	Prescription string
	Severity     string
}

type ConditionCollection map[string]*Condition

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
		Aliases:     []string{"doctor:list"},
		Usage:       "Learn all the rules applied by the doctor:diagnose command.",
		Description: "Display all the conditions for which the doctor:diagnose command will check.",
		Before:      cmd.Before,
		Action:      cmd.RunCompendium,
	}

	return []cli.Command{diagnose, compendium}
}

// RunAnalysis controls the doctor/diagnosis process.
func (cmd *ProjectDoctor) RunAnalysis(ctx *cli.Context) error {
	fmt.Println("Project doctor evaluates project-specific environment issues.")
	fmt.Println("You will find most of the checks defined in your Outrigger Project configuration (e.g., outrigger.yml)")
	fmt.Println("These checks are not comprehensive, this is intended to automate common environment troubleshooting steps.")
	fmt.Println()
	compendium, _ := cmd.GetConditionCollection()
	if err := cmd.AnalyzeConditionList(compendium); err != nil {
		// Directly returning the framework error to skip the expanded help.
		// A failing state is self-descriptive.
		return cli.NewExitError(fmt.Sprintf("%v", err), 1)
	}

	return nil
}

// RunCompendium lists all conditions to be checked in the analysis.
func (cmd *ProjectDoctor) RunCompendium(ctx *cli.Context) error {
	compendium, _ := cmd.GetConditionCollection()
	cmd.out.Info("There are %d conditions in the repertoire.", len(compendium))
	fmt.Println(compendium)

	return nil
}

// AnalyzeConditionList checks each registered condition against environment state.
func (cmd *ProjectDoctor) AnalyzeConditionList(conditions ConditionCollection) error {
	var returnVal error

	failing := ConditionCollection{}
	for _, condition := range conditions {
		cmd.out.Spin(fmt.Sprintf("Examining project environment for %s", condition.Name))
		if found := cmd.Analyze(condition); !found {
			cmd.out.Info("Not Affected by: %s [%s]", condition.Name, condition.Id)
		} else {
			switch condition.Severity {
			case ConditionSeverityWARNING:
				cmd.out.Warning("Condition Detected: %s [%s]", condition.Name, condition.Id)
				failing[condition.Id] = condition
				break
			case ConditionSeverityERROR:
				cmd.out.Error("Condition Detected: %s [%s]", condition.Name, condition.Id)
				failing[condition.Id] = condition
				if returnVal == nil {
					returnVal = errors.New("Diagnosis found at least one failing condition.")
				}
				break
			default:
				cmd.out.Info("Condition Detected: %s [%s]", condition.Name, condition.Id)
			}
		}
	}

	if len(failing) > 0 {
		color.Red("\nThere were %d problems identified out of %d checked.\n", len(failing), len(conditions))
		fmt.Println(failing)
	}

	return returnVal
}

// GetConditionCollection assembles a list of all conditions.
func (cmd *ProjectDoctor) GetConditionCollection() (ConditionCollection, error) {
	conditions := cmd.Config.Doctor

	// @TODO move these to outrigger.yml once we have pure shell facilities.
	eval := ProjectEval{cmd.out, cmd.Config}
	sync := ProjectSync{}
	syncName := sync.GetVolumeName(cmd.Config, eval.GetWorkingDirectory())

	// @todo we should have a way to determine if the project wants to use sync.
	item1 := &Condition{
		Id:           "sync-container-not-running",
		Name:         "Sync Container Not Working",
		Test:         []string{fmt.Sprintf("$(id=$(docker container ps -aq --filter 'name=^/%s$'); docker top $id &>/dev/null)", syncName)},
		Diagnosis:    "The Sync container for this project is not available.",
		Prescription: "Run 'rig project sync:start' before beginning work. This command may be included in other project-specific tasks.",
		Severity:     ConditionSeverityWARNING,
	}
	if _, ok := conditions["sync-container-not-running"]; !ok {
		conditions["sync-container-not-running"] = item1
	}

	item2 := &Condition{
		Id:           "sync-volume-missing",
		Name:         "Sync Volume is Missing",
		Test:         []string{fmt.Sprintf("$(id=$(docker container ps -aq --filter 'name=^/%s$'); docker top $id &>/dev/null)", syncName)},
		Diagnosis:    "The Sync volume for this project is missing.",
		Prescription: "Run 'rig project sync:start' before beginning work. This command may be included in other project-specific tasks.",
		Severity:     ConditionSeverityWARNING,
	}
	if _, ok := conditions["sync-volume-missing"]; !ok {
		conditions["sync-volume-missing"] = item2
	}

	return conditions, nil
}

// Analyze if a given condition criteria is met.
func (cmd *ProjectDoctor) Analyze(c *Condition) bool {
	eval := ProjectEval{cmd.out, cmd.Config}
	script := &ProjectScript{c.Id, "", c.Name, c.Test}

	if _, exitCode, err := eval.ProjectScriptResult(script, []string{}); err != nil {
		cmd.out.Verbose("Condition '%s' analysis failed: (%d)", c.Id, exitCode)
		cmd.out.Verbose("Error: %s", err.Error())
		return true
	}

	return false
}

// String converts a ConditionCollection to a string.
// @TODO use a good string concatenation technique, unlike this.
func (cc ConditionCollection) String() string {
	str := ""
	for _, condition := range cc {
		str = fmt.Sprintf(fmt.Sprintf("%s\n%s\n", str, condition))
	}
	return fmt.Sprintf(fmt.Sprintf("%s\n", str))
}

// String converts a Condition to a string.
func (c Condition) String() string {
	return fmt.Sprintf("%s (%s)\n\tDESCRIPTION: %s\n\tSOLUTION: %s\n\t[%s]",
		headline(c.Name),
		severityFormat(c.Severity),
		c.Diagnosis,
		c.Prescription,
		c.Id)
}

func headline(value string) string {
	h := color.New(color.Bold, color.Underline).SprintFunc()
	return h(value)
}

func severityFormat(severity string) string {

	switch severity {
	case ConditionSeverityWARNING:
		yellow := color.New(color.FgYellow).SprintFunc()
		return yellow(strings.ToUpper(severity))
	case ConditionSeverityERROR:
		red := color.New(color.FgRed).SprintFunc()
		return red(strings.ToUpper(severity))
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	return cyan(strings.ToUpper(severity))
}

// SeverityList supplies the valid conditions as an array.
func SeverityList() []string {
	return []string{ConditionSeverityINFO, ConditionSeverityWARNING, ConditionSeverityERROR}
}
