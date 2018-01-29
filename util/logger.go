package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/fatih/color"
	spun "github.com/slok/gospinner"
)

// logger is the global logger data structure. Retrieve via Logger().
var logger *RigLogger

// logChannels defines various log channels. This nests within the RigLogger to expose the loggers directly for
// advanced use cases.
type logChannels struct {
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
	Verbose *log.Logger
}

// RigLogger is the global logger object
type RigLogger struct {
	Channel   logChannels
	Progress  *RigSpinner
	IsVerbose bool
	Spinning  bool
	Privileged bool
}

// RigSpinner object wrapper to facilitate our spinner service
// as a different
type RigSpinner struct {
	Spins *spun.Spinner
}

// LoggerInit initializes the global logger
func LoggerInit(verbose bool) {
	var verboseWriter = ioutil.Discard
	if verbose {
		verboseWriter = os.Stdout
	}

	s, _ := spun.NewSpinner(spun.Dots)
	logger = &RigLogger{
		Channel: logChannels{
			Info:    log.New(os.Stdout, color.BlueString("[INFO] "), 0),
			Warning: log.New(os.Stdout, color.YellowString("[WARN] "), 0),
			Error:   log.New(os.Stderr, color.RedString("[ERROR] "), 0),
			Verbose: log.New(verboseWriter, "[VERBOSE] ", 0),
		},
		IsVerbose: verbose,
		Progress:  &RigSpinner{s},
		Spinning:  false,
		Privileged: false,
	}
}

// Logger returns the instance of the global logger
func Logger() *RigLogger {
	if logger == nil {
		LoggerInit(false)
	}

	return logger
}

// Spin restarts the spinner for a new task.
func (log *RigLogger) Spin(message string) {
	if !log.IsVerbose {
		log.Progress.Spins.Start(message)
		log.Spinning = true
	}
}

// NoSpin stops the Progress spinner.
func (log *RigLogger) NoSpin() {
	log.Progress.Spins.Stop()
	log.Spinning = false
}

// Info indicates success behavior of the spinner-associated task.
func (log *RigLogger) Info(format string, a ...interface{}) {
	if log.IsVerbose || !log.Spinning {
		log.Channel.Info.Println(fmt.Sprintf(format, a...))
	} else {
		log.Progress.Spins.SetMessage(fmt.Sprintf(format, a...))
		log.Progress.Spins.Succeed()
	}
}

// Warning indicates a warning in the resolution of the spinner-associated task.
func (log *RigLogger) Warning(format string, a ...interface{}) {
	if log.IsVerbose || !log.Spinning {
		log.Channel.Warning.Println(fmt.Sprintf(format, a...))
	} else {
		log.Progress.Spins.SetMessage(fmt.Sprintf(format, a...))
		log.Progress.Spins.Warn()
	}
}

// Warn is a convenience wrapper for Warning.
func (log *RigLogger) Warn(format string, a ...interface{}) {
	log.Warning(format, a...)
}

// Error indicates an error in the spinner-associated task.
func (log *RigLogger) Error(format string, a ...interface{}) {
	if log.IsVerbose || !log.Spinning {
		log.Channel.Error.Println(fmt.Sprintf(format, a...))
	} else {
		log.Progress.Spins.SetMessage(fmt.Sprintf(format, a...))
		log.Progress.Spins.Fail()
	}
}

// Verbose allows Verbose logging of more advanced activities/information.
// In practice, if the spinner can be in use verbose is a no-op.
func (log *RigLogger) Verbose(format string, a ...interface{}) {
	log.Channel.Verbose.Println(fmt.Sprintf(format, a...))
}

// Note allows output of an info log, bypassing the spinner if in use.
func (log *RigLogger) Note(format string, a ...interface{}) {
	log.Channel.Info.Println(fmt.Sprintf(format, a...))
}

// PrivilegeEscallationPrompt interrupts a running spinner to ensure clear
// prompting to the user for sudo password entry. It is up to the caller to know
// that privilege is needed, but this method will track state to determine if
// a privilege escallation previously happened, and assume the user successfully
// entered their password.
func (log *RigLogger) PrivilegeEscallationPrompt() {
	defer func() { log.Privileged = true }()

	if log.Privileged {
		return
	}

	log.NoSpin()
	fmt.Println()

	// After some time exploring the various options with the Spinner library,
	// this kind of rapid manipulation of the spinner without other operations
	// seems to be confounding the go routines, breaking the intended "UI".
	// Hacking timing workarounds to force spinner processing sequencing help.
	time.Sleep(10 * time.Millisecond)
	log.Spin("Evaluating administrative action...")
	log.Warning("Administrative privileges needed, please enter your password:")
	time.Sleep(30 * time.Millisecond)
}
