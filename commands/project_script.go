package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/phase2/rig/util"
)

// ProjectScript wraps the evaluation of project scripts.
// It mimics command struct except with unexported values.
type ProjectScript struct {
	out    *util.RigLogger
	config *ProjectConfig
}

// Run takes a Script configuration and executes it per the definition of
// the project script and bonus arguments from the extra parameter.
// Commands are run from the directory context of the project if available.
// This also supports follow-up user interaction.
func (p *ProjectScript) Run(script *Script, extra []string) int {
	return util.PassthruCommand(p.prepareToExecute(script, extra))
}

// Capture matches Run, but returns the data from the command
// execution instead of "streaming" the result to the terminal.
func (p *ProjectScript) Capture(script *Script, extra []string) (string, int, error) {
	return util.CaptureCommand(p.prepareToExecute(script, extra))
}

// prepareToExecute is an internal method that handles standardized "preflight"
// steps before executing the command, including logging.
func (p *ProjectScript) prepareToExecute(script *Script, extra []string) *exec.Cmd {
	p.out.Verbose("Initializing project script '%s': %s", script.ID, script.Description)
	p.addCommandPath()
	dir := p.GetWorkingDirectory()
	shellCmd := p.CreateCommand(script.Run, extra, dir)
	shellCmd.Env = append(os.Environ(), "RIG_POWER_USER_MODE=1")
	p.out.Verbose("Evaluating Script '%s'", script.ID)
	return shellCmd
}

// GetCommand is a deprecation wrapper around NormalizeCommand.
// It was renamed for improved clarity alongside other methods.
func (p *ProjectScript) GetCommand(steps, extra []string, workingDirectory string) *exec.Cmd {
	return p.CreateCommand(steps, extra, workingDirectory)
}

// CreateCommand is a factory method to assemble an executable command from
// project-derived parameters.
// @see https://github.com/medhoover/gom/blob/staging/config/command.go
func (p *ProjectScript) CreateCommand(steps, extra []string, workingDirectory string) *exec.Cmd {
	// Concat the commands together adding the args to this command as args to the last step
	scriptCommands := strings.Join(steps, p.getCommandSeparator()) + " " + strings.Join(extra, " ")

	var command *exec.Cmd
	if util.IsWindows() {
		/* #nosec */
		command = exec.Command("cmd", "/c", scriptCommands)
	} else {
		/* #nosec */
		command = exec.Command("sh", "-c", scriptCommands)
	}
	command.Dir = workingDirectory

	return command
}

// GetWorkingDirectory retrieves the working directory for project commands.
func (p *ProjectScript) GetWorkingDirectory() string {
	return filepath.Dir(p.config.Path)
}

// getCommandSeparator returns the command separator based on platform.
func (p *ProjectScript) getCommandSeparator() string {
	if util.IsWindows() {
		return " & "
	}

	return " && "
}

// addCommandPath overrides the PATH environment variable for further shell executions.
// This is used on POSIX systems for lookup of scripts.
func (p *ProjectScript) addCommandPath() {
	binDir := p.config.Bin
	if binDir != "" {
		p.out.Verbose("Adding project bin directory to $PATH: %s", binDir)
		path := os.Getenv("PATH")
		os.Setenv("PATH", fmt.Sprintf("%s%c%s", binDir, os.PathListSeparator, path))
	}
}
