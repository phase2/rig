package util

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"

	"github.com/fatih/color"
)

const defaultFailedCode = 1

// Set up the output streams (and colors) to stream command output if verbose is configured
func StreamCommand(cmd *exec.Cmd) error {
	return RunCommand(cmd, false)
}

// Set up the output streams (and colors) to stream command output regardless of verbosity
func ForceStreamCommand(cmd *exec.Cmd) error {
	return RunCommand(cmd, true)
}

// Run the command
func RunCommand(cmd *exec.Cmd, forceOutput bool) error {
	cmd.Stderr = os.Stderr
	if Logger().IsVerbose || forceOutput {
		cmd.Stdout = os.Stdout
	} else {
		cmd.Stdout = ioutil.Discard
	}

	color.Set(color.FgCyan)
	err := cmd.Run()
	color.Unset()
	return err
}

// This is similar to ForceStreamCommand in that it will issue all output
// regardless of verbose mode. Further, this version of the command captures the
// exit status of any executed command. This function is intended to simulate
// native execution of the command passed to it.
//
// @todo streaming the output instead of buffering until completion.
func PassthruCommand(cmd *exec.Cmd) (stdout string, stderr string, exitCode int) {
	var outbuf, errbuf bytes.Buffer

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
        cmd.Stdin = os.Stdin

	err := cmd.Run()
	stdout = outbuf.String()
	stderr = errbuf.String()

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
			if stderr == "" {
				stderr = err.Error()
			}
		}
	} else {
		// Success, exitCode should be 0.
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	return
}
