package notify
/**
 * urfave/cli Notify "Extension"
 *
 * This sub-package of rig is intended for potential separation into a
 * standalone package or PR to the main framework. As such, it should not have
 * any built-in assumptions or identifiers about Rig, nor depend on anything in
 * the rig codebase.
 */

import (
	"fmt"
  "os"
  "strings"

  "github.com/martinlindhe/notify"
  "github.com/urfave/cli"
)

var config *NotifyConfig

type NotifyConfig struct {
  // Holds a list of command names and their default value for notification purposes.
  // This needs to be retained after setting up the flags to more easily determine
  // the notification criteria for each command. Without this, there would be a
  // lot more array iterating.
	Commands  map[string]bool
  // The name of the app for environment variables.
	Id string
  // Relative path to notification logo.
  Icon      string
  // The label for the app to be displayed in notifications.
  Label     string
}

// Initialize notifications.
func Init(id string, label string, icon string, commands map[string]bool) error {
  config = &NotifyConfig{
    Commands:   commands,
		Id:       id,
		Icon:     icon,
    Label:   label,
	}
  return nil
}

// Adds notitification flags to the provided commands.
func AddNotifications(commands []cli.Command) []cli.Command {
  for i, command := range commands {
    if enable, ok := config.Commands[command.Name]; ok {
      commands[i].Flags = append(command.Flags, getNotifyFlag(enable, command.Name))
    }
  }
  return commands
}

// Used to trigger a standardized notification based on the command's
// self-reported success or failure.
func CommandStatus(ctx *cli.Context, success bool) error {
  if commandShouldSignal(ctx) {
    if success {
      Notify(fmt.Sprintf("%s Succeeded", ctx.Command.Name), "The command is complete.")
    } else {
      Notify(fmt.Sprintf("%s Failed", ctx.Command.Name), "The command encountered a problem.")
    }
	}

	return nil
}

// Allow a command to send an arbitrary message if notifications are enabled.
func CommandMessage(ctx *cli.Context, message string) error {
  if commandShouldSignal(ctx) {
		Notify(ctx.Command.Name, message)
	}
	return nil
}

// Send the notification via the Notify library (as opposed to this notify package.)
func Notify(title string, message string) error {
  if len(os.Getenv(getEnvVarName("NOTIFY_SILENCE_ALL"))) == 0 {
    notify.Notify(config.Label, title, message, config.Icon)
  }
  return nil
}

// Determine if a command-triggered notification should be sent.
func commandShouldSignal(ctx *cli.Context) bool {
  enabled, ok := config.Commands[ctx.Command.Name];
  return ok && ((enabled && !ctx.Bool("no-notify")) || (!enabled && !ctx.Bool("notify")))
}

// Generates a cleaned up environment name, all caps and underscores.
func getEnvVarName(suffix string) string {
  prefix := strings.ToUpper(config.Id)
  suffix = strings.ToUpper(suffix)
  joined := fmt.Sprintf("%s_%s", prefix, suffix)
  return strings.Replace(joined, "-", "_", -1)
}

// Generates standard flags for toggling notifications.
// These are generated based on whether it should default to notify or not.
// Flags are expected to be per-command.
func getNotifyFlag(notify bool, namespace string) cli.Flag {
	if notify {
    envVarName := getEnvVarName(fmt.Sprintf("NOTIFY_DEFAULT_DISABLE_%s", namespace))
		return cli.BoolFlag{
			Name:   "no-notify",
			Usage:  "Mute desktop notification.",
			EnvVar: envVarName,
		}
	} else {
    envVarName := getEnvVarName(fmt.Sprintf("NOTIFY_DEFAULT_ENABLE_%s", namespace))
		return cli.BoolFlag{
			Name:   "notify",
			Usage:  "Trigger desktop notification when command completes.",
			EnvVar: envVarName,
		}
	}
}
