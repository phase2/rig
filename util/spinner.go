package util

import (
	base "github.com/slok/gospinner"
)

// RigSpinner object wrapper to facilitate our spinner service
// as a different
type RigSpinner struct {
	Spins *base.Spinner
}

// progress is a global spinner object.
var progress *RigSpinner

// Spinner returns the instance of the global spinner.
func Spinner() *RigSpinner {
	if progress == nil {
		SpinnerInit()
	}

	return progress
}

// SpinnerInit creates a new progress spinner if verbose logging is not enabled and
// the spinner does not yet exist, then starts the spinner.
func SpinnerInit() *RigSpinner {
	s, _ := base.NewSpinner(base.Dots)
	progress = &RigSpinner{Spins: s}
	return progress
}

// Start restarts the spinner for a new task.
// All functions with manage the spinner are contingent on the logger's verbosity status.
// When verbose, we perform basic logging instead of running the spinner.
func (s *RigSpinner) Start(message string) {
	if Logger().IsVerbose {
		Logger().Info.Println(message)
	} else {
		s.Spins.Start(message)
	}
}

// Complete indicates success behavior of the spinner-associated task.
// We do not surface "Succeed" here to avoid confusion with commands/command.go:Success().
func (s *RigSpinner) Complete(message string) {
	if Logger().IsVerbose {
		Logger().Info.Println(message)
	} else {
		s.Spins.SetMessage(message)
		s.Spins.Succeed()
	}
}

// Warn indicates a warning in the resolution of the spinner-associated task.
func (s *RigSpinner) Warn(message string) {
	if Logger().IsVerbose {
		Logger().Warning.Println(message)
	} else {
		s.Spins.SetMessage(message)
		s.Spins.Warn()
	}
}

// Fail indicates an error in the spinner-associated task.
func (s *RigSpinner) Fail(message string) {
	if Logger().IsVerbose {
		Logger().Error.Println(message)
	} else {
		s.Spins.SetMessage(message)
		s.Spins.Fail()
	}
}
