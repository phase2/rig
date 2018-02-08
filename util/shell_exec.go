package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

const defaultFailedCode = 1

// Executor wraps exec.Cmd to allow consistent manipulation of executed commands.
type Executor struct {
	cmd *exec.Cmd
}

// StreamCommand sets up the output streams (and colors) to stream command output if verbose is configured
func StreamCommand(path string, arg ...string) error {
	return Command(path, arg...).Execute(false)
}

// ForceStreamCommand sets up the output streams (and colors) to stream command output regardless of verbosity
func ForceStreamCommand(path string, arg ...string) error {
	return Command(path, arg...).Execute(true)
}

// Command creates a new Executor instance from the execution arguments.
func Command(path string, arg ...string) Executor {
	/* #nosec */
	return Executor{exec.Command(path, arg...)}
}

// Convert takes a exec.Cmd pointer and wraps it in an executor object.
func Convert(cmd *exec.Cmd) Executor {
	return Executor{cmd}
}

// EscalatePrivilege attempts to gain administrative privilege
// @todo identify administrative escallation on Windows.
// E.g., "runas", "/noprofile", "/user:Administrator
func EscalatePrivilege() error {
	return Command("sudo", "-v").Run()
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

	bin := Executor{cmd}
	err := bin.Run()

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

// Execute executes the provided command, it also can sspecify if the output should be forced to print to the console
func (x Executor) Execute(forceOutput bool) error {
	x.cmd.Stderr = os.Stderr
	if Logger().IsVerbose || forceOutput {
		x.cmd.Stdout = os.Stdout
	} else {
		x.cmd.Stdout = ioutil.Discard
	}

	color.Set(color.FgCyan)
	err := x.Run()
	color.Unset()
	return err
}

// CombinedOutput runs a command via exec.CombinedOutput() without modification or output of the underlying command.
func (x Executor) CombinedOutput() ([]byte, error) {
	x.Log("Executing")
	if out := Logger(); out != nil && x.IsPrivileged() {
		out.PrivilegeEscallationPrompt()
		defer out.Spin("Resuming operation...")
	}
	return x.cmd.CombinedOutput()
}

// Run runs a command via exec.Run() without modification or output of the underlying command.
func (x Executor) Run() error {
	x.Log("Executing")
	if out := Logger(); out != nil && x.IsPrivileged() {
		out.PrivilegeEscallationPrompt()
		defer out.Spin("Resuming operation...")
	}
	return x.cmd.Run()
}

// Output runs a command via exec.Output() without modification or output of the underlying command.
func (x Executor) Output() ([]byte, error) {
	x.Log("Executing")
	if out := Logger(); out != nil && x.IsPrivileged() {
		out.PrivilegeEscallationPrompt()
		defer out.Spin("Resuming operation...")
	}
	return x.cmd.Output()
}

// Start runs a command via exec.Start() without modification or output of the underlying command.
func (x Executor) Start() error {
	x.Log("Executing")
	if out := Logger(); out != nil && x.IsPrivileged() {
		out.PrivilegeEscallationPrompt()
		defer out.Spin("Resuming operation...")
	}
	return x.cmd.Start()
}

// Log verbosely logs the command.
func (x Executor) Log(tag string) {
	color.Set(color.FgMagenta)
	Logger().Verbose("%s: %s", tag, x)
	color.Unset()
}

// String converts a Command to a human-readable string with key context details.
// It is automatically applied in contexts such as fmt functions.
func (x Executor) String() string {
	context := ""
	if x.cmd.Dir != "" {
		context = fmt.Sprintf("(WD: %s", x.cmd.Dir)
	}
	if x.cmd.Env != nil {
		env := strings.Join(x.cmd.Env, " ")
		if context == "" {
			context = fmt.Sprintf("(Env: %s", env)
		} else {
			context = fmt.Sprintf("%s, Env: %s)", context, env)
		}
	}

	return fmt.Sprintf("%s %s %s", x.cmd.Path, strings.Join(x.cmd.Args[1:], " "), context)
}

// IsPrivileged evaluates the command to determine if administrative privilege
// is required.
// @todo identify administrative escallation on Windows.
// E.g., "runas", "/noprofile", "/user:Administrator
func (x Executor) IsPrivileged() bool {
	_, privileged := IndexOfSubstring(x.cmd.Args, "sudo")
	return privileged
}
