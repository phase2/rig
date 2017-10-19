package commands

import (
	"flag"
	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type RigCommand interface {
	Commands() []cli.Command
}

type BaseCommand struct {
	RigCommand
	out     *util.RigLogger
	machine Machine
}

// Run before all commands to setup core services.
func (cmd *BaseCommand) Before(c *cli.Context) error {
	// Re-initialize logger in case Commands() call led to logger usage which
	// initialized the logger without the verbose flag if present.
	util.LoggerInit(c.GlobalBool("verbose"))
	cmd.out = util.Logger()
	cmd.machine = Machine{Name: c.GlobalString("name"), out: util.Logger()}
	return nil
}

func (cmd *BaseCommand) NewContext(name string, flags []cli.Flag, parent *cli.Context) *cli.Context {
	flagSet := flag.NewFlagSet(name, flag.ContinueOnError)
	for _, f := range flags {
		f.Apply(flagSet)
	}
	return cli.NewContext(parent.App, flagSet, parent)
}

func (cmd *BaseCommand) SetContextFlag(ctx *cli.Context, name string, value string) {
	if err := ctx.Set(name, value); err != nil {
		cmd.out.Error.Fatal(err)
	}
}
