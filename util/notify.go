package util

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

	"github.com/martinlindhe/notify"
	"github.com/urfave/cli"
)

var config *NotifyConfig

// NotifyConfig holds configuration for notification support
type NotifyConfig struct {
	// The label for the app to be displayed in notifications.
	Label string

	// Relative path to notification logo.
	Icon string
}

// NotifyInit initializes notification config
func NotifyInit(label string) error {
	config = &NotifyConfig{
		Icon:  "util/logo.png",
		Label: label,
	}
	return nil
}

// NotifySuccess send a notification for a successful command run
func NotifySuccess(ctx *cli.Context, message string) {
	if shouldNotify(ctx) {
		notify.Notify(config.Label, fmt.Sprintf("Success: %s", ctx.Command.Name), message, config.Icon)
	}
}

// NotifyError send a notification for a failed command run
func NotifyError(ctx *cli.Context, message string) error {
	if shouldNotify(ctx) {
		notify.Notify(config.Label, fmt.Sprintf("Error: %s", ctx.Command.Name), message, config.Icon)
	}
	return nil
}

// shouldNotify returns a boolean if notifications are enabled
func shouldNotify(ctx *cli.Context) bool {
	return !ctx.GlobalBool("quiet")
}
