package commands

import (
	"flag"
	"fmt"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// RigCommand is the interface for all rig commands
type RigCommand interface {
	Commands() []cli.Command
}

// BaseCommand is parent for all rig commands
type BaseCommand struct {
	RigCommand
	out     *util.RigLogger
	machine Machine
	context *cli.Context
}

// Before configure the function to run before all commands to setup core services.
func (cmd *BaseCommand) Before(c *cli.Context) error {
	// Re-initialize logger in case Commands() call led to logger usage which
	// initialized the logger without the verbose flag if present.
	util.LoggerInit(c.GlobalBool("verbose"))
	util.ToggleNoOpMode(c.GlobalBool("simulate"))
	cmd.out = util.Logger()
	cmd.machine = Machine{Name: c.GlobalString("name"), out: util.Logger()}

	util.NotifyInit(fmt.Sprintf("Outrigger (rig) %s", c.App.Version))

	// Hold onto Context so that we can use it later without having to pass it around everywhere
	cmd.context = c

	return nil
}

// Success encapsulates the functionality for reporting command success
func (cmd *BaseCommand) Success(message string) error {
	// Handle success messaging.
	if message != "" {
		cmd.out.Info(message)
		util.NotifySuccess(cmd.context, message)
	}

	// If there is an active spinner wrap it up. This is not placed before the logging above so commands can rely on
	// cmd.Success to set the last spinner status in lieu of an extraneous log entry.
	cmd.out.NoSpin()

	return nil
}

// Failure encapsulates the functionality for reporting command failure
func (cmd *BaseCommand) Failure(message string, errorName string, exitCode int) error {
	// Make sure any running spinner halts.
	cmd.out.NoSpin()
	// Handle error messaging.
	util.NotifyError(cmd.context, message)

	return cli.NewExitError(fmt.Sprintf("ERROR: %s [%s] (%d)", message, errorName, exitCode), exitCode)
}

// NewContext creates a new Context struct to pass along to delegate commands
func (cmd *BaseCommand) NewContext(name string, flags []cli.Flag, parent *cli.Context) *cli.Context {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(flagSet)
	}
	return cli.NewContext(parent.App, flagSet, parent)
}

// SetContextFlag set a flag on the provided context
func (cmd *BaseCommand) SetContextFlag(ctx *cli.Context, name string, value string) {
	if err := ctx.Set(name, value); err != nil {
		cmd.out.Channel.Error.Fatal(err)
	}
}
