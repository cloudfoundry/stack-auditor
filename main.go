package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/cloudfoundry/stack-auditor/utils"

	"github.com/cloudfoundry/stack-auditor/terminalUI"

	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/cloudfoundry/stack-auditor/changer"
	"github.com/cloudfoundry/stack-auditor/deleter"

	"github.com/cloudfoundry/stack-auditor/auditor"

	"code.cloudfoundry.org/cli/plugin"
)

type StackAuditor struct {
	UI terminalUI.UIController
}

const (
	AuditStackCmd    = "audit-stack"
	ChangeStackCmd   = "change-stack"
	DeleteStackCmd   = "delete-stack"
	ChangeStackUsage = "Usage: cf change-stack <app> <stack> [--v3]"
	V3Flag           = "--v3"
)

func main() {
	stackAuditor := StackAuditor{
		UI: terminalUI.NewUi(),
	}
	plugin.Start(&stackAuditor)
}

func (s *StackAuditor) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 {
		err := errors.New("no command line arguments provided")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	switch args[0] {
	case AuditStackCmd:
		a := auditor.Auditor{
			CF: cf.CF{
				Conn: cliConnection,
			},
		}
		info, err := a.Audit()
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}
		fmt.Println(info)

	case DeleteStackCmd:
		forceFlag := len(args) > 2 && (args[2] == "--force" || args[2] == "-f")

		if !forceFlag && !s.UI.ConfirmDelete(args[1]) {
			os.Exit(1)
		}

		a := deleter.Deleter{
			CF: cf.CF{
				Conn: cliConnection,
			},
		}
		info, err := a.DeleteStack(args[1])
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}
		fmt.Println(info)

	case ChangeStackCmd:
		if len(args) != 3 && len(args) != 4 {
			log.Fatalf("Incorrect number of arguments provided - %s\n", ChangeStackUsage)
		}

		c := changer.Changer{}
		c.Runner = utils.Command{}
		c.CF = cf.CF{
			Conn: cliConnection,
		}

		info, err := c.ChangeStack(args[1], args[2])
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}
		fmt.Println(info)

	case "CLI-MESSAGE-UNINSTALL":
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Unknown argument provided")
		os.Exit(17)
	}
}

func (s *StackAuditor) GetMetadata() plugin.PluginMetadata {
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
				Name:     AuditStackCmd,
				HelpText: "List all apps with their stacks, orgs, and spaces",

				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf("cf %s", AuditStackCmd),
				},
			},
			{
				Name:     DeleteStackCmd,
				HelpText: "Delete a stack from the foundation",

				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf("cf %s STACK_NAME", DeleteStackCmd),
				},
			},
			{
				Name:     ChangeStackCmd,
				HelpText: "Change an app's stack in the current space and restart the app",

				UsageDetails: plugin.Usage{
					Usage: fmt.Sprintf("cf %s APP_NAME STACK_NAME", ChangeStackCmd),
				},
			},
		},
	}
}
