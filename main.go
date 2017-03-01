package main

import (
	"os"

	"github.com/urfave/cli"
)

const VERSION = "1.1.0"

type RigCommand interface {
	Commands() cli.Command
}

var machine Machine

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
		LoggerInit(c.GlobalBool("verbose"))
		machine = Machine{Name: c.GlobalString("name")}
		return nil
	}

	app.Commands = []cli.Command{}
	app.Commands = append(app.Commands, (&Start{}).Commands())
	app.Commands = append(app.Commands, (&Stop{}).Commands())
	app.Commands = append(app.Commands, (&Restart{}).Commands())
	app.Commands = append(app.Commands, (&Upgrade{}).Commands())
	app.Commands = append(app.Commands, (&Status{}).Commands())
	app.Commands = append(app.Commands, (&Config{}).Commands())
	app.Commands = append(app.Commands, (&Dns{}).Commands())
	app.Commands = append(app.Commands, (&DnsRecords{}).Commands())
	app.Commands = append(app.Commands, (&Dashboard{}).Commands())
	app.Commands = append(app.Commands, (&Prune{}).Commands())
	app.Commands = append(app.Commands, (&DataBackup{}).Commands())
	app.Commands = append(app.Commands, (&DataRestore{}).Commands())
	app.Commands = append(app.Commands, (&Kill{}).Commands())
	app.Commands = append(app.Commands, (&Remove{}).Commands())
	app.Commands = append(app.Commands, (&Watch{}).Commands())
	app.Commands = append(app.Commands, (&Doctor{}).Commands())

	app.Run(os.Args)
}
