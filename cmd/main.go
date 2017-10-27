package main

import (
	"os"

	"github.com/phase2/rig/commands"
	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

var version = "master"

// It all starts here
func main() {
	app := cli.NewApp()
	app.Name = "rig"
	app.Usage = "Containerized platform environment for projects"
	app.Version = version
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
	app.Commands = append(app.Commands, (&commands.Project{}).Commands()...)
	app.Commands = append(app.Commands, (&commands.Doctor{}).Commands()...)

	app.Run(os.Args)
}
