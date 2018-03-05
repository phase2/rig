package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/phase2/rig/util"
)

type ProjectEval struct {
	out    *util.RigLogger
	config *ProjectConfig
}

// ProjectScriptRun takes a ProjectScript configuration and executes it per
// the definition of the project script and bonus arguments from the extra parameter.
// Commands are run from the directory context of the project if available.
// Use ProjectScriptRun to run a comman for potential user interaction.
func (p *ProjectEval) ProjectScriptRun(script *ProjectScript, extra []string) int {
	p.out.Verbose("Initializing project script '%s': %s", script.Id, script.Description)
	p.addCommandPath()
	dir := p.GetWorkingDirectory()
	shellCmd := p.GetCommand(script.Run, extra, dir)
	p.out.Verbose("Evaluating Script '%s'", script.Id)
	return util.PassthruCommand(shellCmd)
}

// ProjectScriptResult matches ProjectScriptRun, but returns the data from the
// command execution instead of "streaming" the result to the terminal.
func (p *ProjectEval) ProjectScriptResult(script *ProjectScript, extra []string) (string, int, error) {
	p.out.Verbose("Initializing project script '%s': %s", script.Id, script.Description)
	p.addCommandPath()
	dir := p.GetWorkingDirectory()
	shellCmd := p.GetCommand(script.Run, extra, dir)
	p.out.Verbose("Evaluating Script '%s'", script.Id)
	return util.CaptureCommand(shellCmd)
}

// GetCommand constructs a command to execute a configured script.
// @see https://github.com/medhoover/gom/blob/staging/config/command.go
func (p *ProjectEval) GetCommand(steps, extra []string, workingDirectory string) *exec.Cmd {
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
func (p *ProjectEval) GetWorkingDirectory() string {
	return filepath.Dir(p.config.Path)
}

// getCommandSeparator returns the command separator based on platform.
func (p *ProjectEval) getCommandSeparator() string {
	if util.IsWindows() {
		return " & "
	}

	return " && "
}

// addCommandPath overrides the PATH environment variable for further shell executions.
// This is used on POSIX systems for lookup of scripts.
func (p *ProjectEval) addCommandPath() {
	binDir := p.config.Bin
	if binDir != "" {
		p.out.Verbose("Adding project bin directory to $PATH: %s", binDir)
		path := os.Getenv("PATH")
		os.Setenv("PATH", fmt.Sprintf("%s%c%s", binDir, os.PathListSeparator, path))
	}
}
