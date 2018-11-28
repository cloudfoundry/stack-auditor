package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.cloudfoundry.org/cli/plugin"
)

type Plugin struct{}

func (c *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 {
		err := errors.New("no command line arguments provided")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Ensure that we called the command audit-stack
	switch args[0] {
	case "audit-stack":
		fmt.Println("This command outputs dummy text.")

		exitChan := make(chan struct{})
		signalChan := make(chan os.Signal, 1)

		signal.Notify(make(chan os.Signal), syscall.SIGHUP)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-signalChan
			close(exitChan)
		}()

		timer := time.NewTimer(10 * time.Second)

		select {
		case <-timer.C:
			fmt.Println("10 seconds elapsed")
		case <-exitChan:
			os.Exit(128)
		}
	case "CLI-MESSAGE-UNINSTALL":
		os.Exit(0)
	default:
		os.Exit(17)
	}
}

func (c *Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "StackAuditor",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "audit-stack",
				HelpText: "Audit stack command's help text",

				// UsageDetails is optional
				// It is used to show help of usage of each command
				UsageDetails: plugin.Usage{
					Usage: "audit-stack\n   cf audit-stack",
				},
			},
		},
	}
}

func main() {
	plugin.Start(new(Plugin))
}
