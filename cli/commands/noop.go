package commands

import (
	"github.com/urfave/cli"

	"github.com/phase2/rig/cli/notify"
)

type Noop struct {
	BaseCommand
}

func (cmd *Noop) Commands() []cli.Command {
	return []cli.Command{
		{
			Name:     "noop",
			Usage:    "Test execution of a command via a dummy operation.",
			Hidden:   true,
			HideHelp: true,
			Before:   cmd.Before,
			Action:   cmd.Run,
		},
	}
}

func (cmd *Noop) Run(ctx *cli.Context) error {
	cmd.out.Info.Println("No-Op Command Executed.")
	notify.CommandStatus(ctx, true)
	notify.CommandMessage(ctx, "No-Op doesn't do anything.")
	return nil
}
