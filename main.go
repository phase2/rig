package main

import (
	"log"
	"os"
	"io"
	"io/ioutil"

	"github.com/fatih/color"
	"github.com/urfave/cli"
)

const VERSION = "1.1.0"

type RigLogger struct {
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
	Verbose *log.Logger
}

type RigCommand interface {
	Commands() cli.Command
}

var out RigLogger;
var verboseWriter io.Writer;

var machine Machine

func LoggerInit(
  infoHandle io.Writer,
  warningHandle io.Writer,
  errorHandle io.Writer,
	verboseHandle io.Writer) {

  out = RigLogger{
		Info:    log.New(infoHandle, color.BlueString("[INFO] "), 0),
		Warning: log.New(warningHandle, color.YellowString("[WARN] "), 0),
		Error:   log.New(errorHandle, color.RedString("[ERROR] "), 0),
		Verbose: log.New(verboseHandle, "[VERBOSE] ", 0),
  }
}

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
			Name: "verbose",
			Usage: "Show verbose output. Learning Mode!",
			EnvVar: "RIG_VERBOSE",
		},
	}

	app.Before = func(c *cli.Context) error {
		verboseWriter = ioutil.Discard
		if c.GlobalBool("verbose"){
			verboseWriter = os.Stdout
		}
		LoggerInit(os.Stdout, os.Stdout, os.Stderr, verboseWriter)

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
