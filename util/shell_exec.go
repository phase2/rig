package util

import (
	"io/ioutil"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

const defaultFailedCode = 1

// StreamCommand sets up the output streams (and colors) to stream command output if verbose is configured
func StreamCommand(cmd *exec.Cmd) error {
	return RunCommand(cmd, false)
}

// ForceStreamCommand sets up the output streams (and colors) to stream command output regardless of verbosity
func ForceStreamCommand(cmd *exec.Cmd) error {
	return RunCommand(cmd, true)
}

// RunCommand executes the provided command, it also can sspecify if the output should be forced to print to the console
func RunCommand(cmd *exec.Cmd, forceOutput bool) error {
	cmd.Stderr = os.Stderr
	if Logger().IsVerbose || forceOutput {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = ioutil.Discard
	}

	color.Set(color.FgCyan)

	err := Run(cmd)
	color.Unset()
	return err
}

// Run provides a wrapper to os/exec.Run() that verbose logs the executed command invocation.
func Run(cmd *exec.Cmd) error {
	Logger().Verbose.Printf("Executing: %s", CmdToString(cmd))
	return cmd.Run()
}

// PassthruCommand is similar to ForceStreamCommand in that it will issue all output
// regardless of verbose mode. Further, this version of the command captures the
// exit status of any executed command. This function is intended to simulate
// native execution of the command passed to it.
//
// Derived from: http://stackoverflow.com/a/40770011/38408
func PassthruCommand(cmd *exec.Cmd) (exitCode int) {
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	if err != nil {
		// Try to get the exit code.
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			exitCode = defaultFailedCode
		}
	} else {
		// Success, exitCode should be 0.
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	return
}

// CmdToString converts a Command to a human-readable string with key context details.
func CmdToString(cmd *exec.Cmd) string {
	context := ""
	if cmd.Dir != "" {
		context = fmt.Sprintf("(WD: %s", cmd.Dir)
	}
	if cmd.Env != nil {
		env := strings.Join(cmd.Env, " ")
		if context == "" {
			context = fmt.Sprintf("(Env: %s", env)
		} else {
			context = fmt.Sprintf("%s, Env: %s)", context, env)
		}
	}

	return fmt.Sprintf("%s %s %s", cmd.Path, strings.Join(cmd.Args[1:], " "), context)
}
