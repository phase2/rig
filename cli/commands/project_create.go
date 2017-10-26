package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/phase2/rig/cli/util"
	"github.com/urfave/cli"
)

type ProjectCreate struct {
	BaseCommand
}

func (cmd *ProjectCreate) Commands() []cli.Command {
	create := cli.Command{
		Name:        "create",
		Aliases:     []string{},
		Usage:       "Run a code generator to generate scaffolding for a new project.",
		ArgsUsage:   "[optional type] [optional args]",
		Description: "The type is the generator to run with args passed to that generator. If using flag arguments use -- before specifying type and arguments.",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "image",
				Usage:  "Docker image to use if default outrigger/generator is not desired.",
				EnvVar: "RIG_PROJECT_CREATE_IMAGE",
			},
			cli.BoolFlag{
				Name:   "no-update",
				Usage:  "Prevent automatic update of designated generator docker image.",
				EnvVar: "RIG_PROJECT_CREATE_NO_UPDATE",
			},
		},
		Before: cmd.Before,
		Action: cmd.Create,
	}

	return []cli.Command{create}
}

// Run a docker image to execute the desired generator
func (cmd *ProjectCreate) Create(ctx *cli.Context) error {
	image := ctx.String("image")
	if image == "" {
		image = "outrigger/generator"
	}

	argsMessage := " with no arguments"
	if ctx.Args().Present() {
		argsMessage = fmt.Sprintf(" with arguments: %s", strings.Join(ctx.Args(), " "))
	}

	if cmd.machine.IsRunning() || runtime.GOOS == "linux" {
		cmd.out.Verbose.Printf("Executing container %s%s", image, argsMessage)
		if err := cmd.RunGenerator(ctx, cmd.machine, image); err != nil {
			return err
		}
	} else {
		return cmd.Error(fmt.Sprintf("Machine '%s' is not running.", cmd.machine.Name), "MACHINE-STOPPED", 12)
	}

	return cmd.Success("")
}

func (cmd *ProjectCreate) RunGenerator(ctx *cli.Context, machine Machine, image string) error {
	machine.SetEnv()

	// The check for whether the image is older than 30 days is not currently used.
	_, seconds, err := util.ImageOlderThan(image, 86400*30)
	if err == nil {
		cmd.out.Verbose.Printf("Local copy of the image '%s' was originally published %0.2f days ago.", image, seconds/86400)
	}

	// If there was an error it implies no previous instance of the image is available
	// or that docker operations failed and things will likely go wrong anyway.
	if err == nil && !ctx.Bool("no-update") {
		cmd.out.Verbose.Printf("Attempting to update %s", image)
		if err := util.StreamCommand(exec.Command("docker", "pull", image)); err != nil {
			cmd.out.Verbose.Println("Failed to update generator image. Will use local cache if available.")
		}
	} else if err == nil && ctx.Bool("no-update") {
		cmd.out.Verbose.Printf("Automatic generator image update suppressed by --no-update option.")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return cmd.Error(fmt.Sprintf("Couldn't determine current working directory: %s", err), "WORKING-DIR-NOT-FOUND", 12)
	}

	// Keep passed in args as distinct elements or they will be treated as
	// a single argument containing spaces when the container gets them.
	args := []string{
		"container",
		"run",
		"--rm",
		"-it",
		"-v", fmt.Sprintf("%s:/generated", cwd),
		image,
	}

	args = append(args, ctx.Args()...)

	shellCmd := exec.Command("docker", args...)
	if exitCode := util.PassthruCommand(shellCmd); exitCode != 0 {
		return cmd.Error(fmt.Sprintf("Error running generator %s %s", image, strings.Join(ctx.Args(), " ")), "COMMAND-ERROR", exitCode)
	}

	return nil
}
