package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/phase2/rig/util"
	"github.com/urfave/cli"
)

// ProjectCreate is the command for running the project generator to scaffold a new project
type ProjectCreate struct {
	BaseCommand
}

// Commands returns the operations supported by this command
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

// Create executes the `rig project create` command to execute the desired generator
func (cmd *ProjectCreate) Create(ctx *cli.Context) error {
	image := ctx.String("image")
	if image == "" {
		image = "outrigger/generator"
	}

	argsMessage := " with no arguments"
	if ctx.Args().Present() {
		argsMessage = fmt.Sprintf(" with arguments: %s", strings.Join(ctx.Args(), " "))
	}

	if cmd.machine.IsRunning() || util.SupportsNativeDocker() {
		cmd.out.Error("Executing container %s%s", image, argsMessage)
		if err := cmd.RunGenerator(ctx, cmd.machine, image); err != nil {
			return err
		}
	} else {
		return cmd.Failure(fmt.Sprintf("Machine '%s' is not running.", cmd.machine.Name), "MACHINE-STOPPED", 12)
	}

	return cmd.Success("")
}

// RunGenerator runs the generator image
func (cmd *ProjectCreate) RunGenerator(ctx *cli.Context, machine Machine, image string) error {
	machine.SetEnv()

	// The check for whether the image is older than 30 days is not currently used.
	_, seconds, err := util.ImageOlderThan(image, 86400*30)
	if err == nil {
		cmd.out.Verbose("Local copy of the image '%s' was originally published %0.2f days ago.", image, seconds/86400)
	}

	// If there was an error it implies no previous instance of the image is available
	// or that docker operations failed and things will likely go wrong anyway.
	if err == nil && !ctx.Bool("no-update") {
		cmd.out.Spin(fmt.Sprintf("Attempting to update project generator docker image: %s", image))
		if e := util.StreamCommand("docker", "pull", image); e != nil {
			cmd.out.Error("Project generator docker image failed to update. Using local cache if available: %s", image)
		} else {
			cmd.out.Info("Project generator docker image is up-to-date: %s", image)
		}
	} else if err == nil && ctx.Bool("no-update") {
		cmd.out.Verbose("Automatic generator image update suppressed by --no-update option.")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return cmd.Failure(fmt.Sprintf("Couldn't determine current working directory: %s", err), "WORKING-DIR-NOT-FOUND", 12)
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
	/* #nosec */
	shellCmd := exec.Command("docker", args...)
	if exitCode := util.PassthruCommand(shellCmd); exitCode != 0 {
		return cmd.Failure(fmt.Sprintf("Failure running generator %s %s", image, strings.Join(ctx.Args(), " ")), "COMMAND-ERROR", exitCode)
	}

	return nil
}
