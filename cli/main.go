package main

import (
	"os"
	"fmt"

	"github.com/phase2/rig/cli/commands"
	"github.com/phase2/rig/cli/util"
	"github.com/phase2/rig/cli/notify"
	"github.com/urfave/cli"
)

const VERSION = "1.3.1"

// It all starts here
func main() {
	app := cli.NewApp()
	app.Name = "rig"
	app.Usage = "Containerized platform environment for projects"
	app.Version = VERSION
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "name",
			Value:  "dev",
			Usage:  "Name of the VM.",
			EnvVar: "RIG_ACTIVE_MACHINE",
		},
		cli.BoolFlag{
			Name:   "verbose",
			Usage:  "Show verbose output. Learning Mode!",
			EnvVar: "RIG_VERBOSE",
		},
	}

	app.Before = func(c *cli.Context) error {
		util.LoggerInit(c.GlobalBool("verbose"))
		return nil
	}

	app.Commands = []cli.Command{}
	app.Commands = append(app.Commands, (&commands.Start{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Stop{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Restart{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Upgrade{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Status{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Config{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Dns{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.DnsRecords{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Dashboard{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Prune{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.DataBackup{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.DataRestore{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Kill{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Remove{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Watch{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Project{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Doctor{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Noop{}).Commands()...)


	notificationTitle := fmt.Sprintf("Outrigger CLI (rig) v%s", VERSION)
	notify.Init(app.Name, notificationTitle, "images/logo.png", map[string]bool{
		"start": true,
		"stop": false,
		"restart": true,
		"upgrade": true,
		"dns": false,
		"dashboard": false,
		"data-backup": true,
		"data-restore": true,
		"kill": true,
		"remove": false,
		"project": false,
		"doctor": false,
		"noop": true,
	})

	// Adds --notify or --no-notify flag to each command according to the whitelist.
	// This would be better as part of the command configuration but there does not
	// appear to be an option for arbitrary properties.
	// * Commands marked as true will default to have desktop notifications and a
	// --no-notify flag.
	// * Commands marked as false will default to not have notifications and have
	// a --notify flag to trigger a notification.
	// * Commands excluded will have not trigger notifications or have flags.
	app.Commands = notify.AddNotifications(app.Commands)

	app.Run(os.Args)
}
