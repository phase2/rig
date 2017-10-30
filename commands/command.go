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
	cmd.out = util.Logger()
	cmd.machine = Machine{Name: c.GlobalString("name"), out: util.Logger()}

	util.NotifyInit(fmt.Sprintf("Outrigger (rig) %s", c.App.Version))
	cmd.context = c

	return nil
}

// Success encapsulates the functionality for reporting command success
func (cmd *BaseCommand) Success(message string) error {
	if message != "" {
		cmd.out.Info.Println(message)
		util.NotifySuccess(cmd.context, message)
	}
	return nil
}

// Error encapsulates the functionality for reporting command failure
func (cmd *BaseCommand) Error(message string, errorName string, exitCode int) error {
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
		cmd.out.Error.Fatal(err)
	}
}
