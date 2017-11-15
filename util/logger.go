package util

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
	spun "github.com/slok/gospinner"
)

var logger *RigLogger

// RigLogger is the global logger object
type RigLogger struct {
	Info      *log.Logger
	Warning   *log.Logger
	Error     *log.Logger
	Verbose   *log.Logger
	Message   *log.Logger
	Progress  *RigSpinner
	IsVerbose bool
	Spinning  bool
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
		Info:      log.New(os.Stdout, color.BlueString("[INFO] "), 0),
		Warning:   log.New(os.Stdout, color.YellowString("[WARN] "), 0),
		Error:     log.New(os.Stderr, color.RedString("[ERROR] "), 0),
		Verbose:   log.New(verboseWriter, "[VERBOSE] ", 0),
		Message:   log.New(os.Stdout, " - ", 0),
		IsVerbose: verbose,
		Progress:  &RigSpinner{s},
		Spinning:  false,
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

// Success indicates success behavior of the spinner-associated task.
func (log *RigLogger) Success(message string) {
	if log.IsVerbose || !log.Spinning {
		log.Info.Println(message)
	} else {
		log.Progress.Spins.SetMessage(message)
		log.Progress.Spins.Succeed()
	}
}

// Warn indicates a warning in the resolution of the spinner-associated task.
func (log *RigLogger) Warn(message string) {
	if log.IsVerbose || !log.Spinning {
		log.Warning.Println(message)
	} else {
		log.Progress.Spins.SetMessage(message)
		log.Progress.Spins.Warn()
	}
}

// Error indicates an error in the spinner-associated task.
func (log *RigLogger) Oops(message string) {
	if log.IsVerbose || !log.Spinning {
		log.Error.Println(message)
	} else {
		log.Progress.Spins.SetMessage(message)
		log.Progress.Spins.Fail()
	}
}

// Status allows output of an info log.
func (log *RigLogger) Status(message string) {
	log.Info.Println(message)
}

// Note allows output of a simple message.
func (log *RigLogger) Note(message string) {
	log.Message.Println(message)
}

// Details allows Verbose logging of more advanced activities/information.
func (log *RigLogger) Details(message string) {
	log.Verbose.Println(message)
}