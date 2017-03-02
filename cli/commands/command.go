package commands

import (
	"github.com/urfave/cli"
	"github.com/phase2/rig/cli/util"
	"flag"
)

type RigCommand interface {
	Commands() cli.Command
}

type BaseCommand struct {
	RigCommand
	out *util.RigLogger
	machine Machine
}

// Run before all commands to setup core services
func (cmd *BaseCommand) Before(c *cli.Context) error {
	cmd.out = util.Logger()
	cmd.machine = Machine{Name: c.GlobalString("name"), out: util.Logger()}
	return nil
}

func (cmd *BaseCommand) NewContext(parent *cli.Context) *cli.Context {
	flagSet := flag.NewFlagSet(cmd.Commands().Name, flag.ContinueOnError)
	for _, f := range cmd.Commands().Flags {
		f.Apply(flagSet)
	}
	return cli.NewContext(parent.App, flagSet, parent)
}

func (cmd *BaseCommand) SetContextFlag(ctx *cli.Context, name string, value string) {
	if err := ctx.Set(name, value); err != nil {
		cmd.out.Error.Fatal(err)
	}
}

