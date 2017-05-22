package commands

import (
	"fmt"
	"os"
	"os/exec"
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
				Name:  "image",
				Usage: "Docker image to use if default outrigger/generator is not desired",
			},
		},
		Before:      cmd.Before,
		Action:      cmd.Create,
	}

	return []cli.Command{create}
}

// Run a docker image to execute the desired generator
func (cmd *ProjectCreate) Create(ctx *cli.Context) error {
	image := ctx.String("image")
	if (image == "") {
		image = "outrigger/generator"
	}

	cwd, err := os.Getwd();
	if (err != nil) {
		cmd.out.Error.Printf("Couldn't determine current working directory: %s", err)
		os.Exit(1)
	}

	argsMessage := " with no arguments"
	if (ctx.Args().Present()) {
		argsMessage = fmt.Sprintf(" with arguments: %s", strings.Join(ctx.Args(), " "));
	}
	cmd.out.Verbose.Printf("Executing container %s%s", image, argsMessage)

	// keep passed in args as distinct elements or they will be treated as
	// a single argument containing spaces when the container gets them
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
		cmd.out.Error.Printf("Error running generator %s %s: %d", image, strings.Join(ctx.Args(), " "), exitCode)
		os.Exit(exitCode)
	}

	return nil
}
