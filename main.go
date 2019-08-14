package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudfoundry/stack-auditor/auditor"
	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/cloudfoundry/stack-auditor/changer"
	"github.com/cloudfoundry/stack-auditor/deleter"
	"github.com/cloudfoundry/stack-auditor/terminalUI"
	"github.com/cloudfoundry/stack-auditor/utils"

	"code.cloudfoundry.org/cli/plugin"
)

var tagVersion = "0.0.5"
var pluginVersion plugin.VersionType

type StackAuditor struct {
	UI terminalUI.UIController
}

func init() {
	var major, minor, patch int

	_, err := fmt.Sscanf(tagVersion, "%d.%d.%d", &major, &minor, &patch)
	if err != nil {
		err := errors.New("problem setting version")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	pluginVersion = plugin.VersionType{
		Major: major,
		Minor: minor,
		Build: patch,
	}
}

const (
	AuditStackCmd      = "audit-stack"
	ChangeStackCmd     = "change-stack"
	DeleteStackCmd     = "delete-stack"
	V3CmdFlag          = "--v3"
	ChangeStackUsage   = "Usage: cf change-stack <app> <stack> [" + V3CmdFlag + "]"
	AuditStackUsage    = "Usage: cf audit-stack [--json | --csv]"
	ErrorMsg           = "a problem occurred: %v\n"
	IncorrectArguments = "Incorrect arguments provided - %s\n"
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

		if len(args) > 2 {
			log.Fatalf(IncorrectArguments, AuditStackUsage)
		}

		if len(args) > 1 && args[1] == "--json" {
			a.OutputType = auditor.JSONFlag
		} else if len(args) > 1 && args[1] == "--csv" {
			a.OutputType = auditor.CSVFlag
		} else if len(args) > 1 {
			log.Fatalf(IncorrectArguments, AuditStackUsage)
		}

		info, err := a.Audit()
		if err != nil {
			log.Fatalf(ErrorMsg, err)
		}
		fmt.Println(info)

	case DeleteStackCmd:
		if len(args) < 2 {
			err := errors.New("Incorrect number of arguments provided - Usage: cf delete-stack <stack>")
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

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
			log.Fatalf(ErrorMsg, err)
		}
		fmt.Println(info)

	case ChangeStackCmd:
		if (len(args) != 3 && len(args) != 4) || (len(args) == 4 && args[3] != V3CmdFlag) {
			log.Fatalf("Incorrect arguments provided - %s\n", ChangeStackUsage)
		}

		c := changer.Changer{
			Log: func(w io.Writer, msg string) {
				w.Write([]byte(msg))
			},
		}

		v3Flag := len(args) > 3 && (args[3] == V3CmdFlag)
		if v3Flag {
			c.V3Flag = true
		}

		c.Runner = utils.Command{}

		c.CF = cf.CF{
			Conn: cliConnection,
		}
		space, err := c.CF.Conn.GetCurrentSpace()
		if err != nil {
			log.Fatalf(ErrorMsg, err)
		}
		c.CF.Space = space

		info, err := c.ChangeStack(args[1], args[2])
		if err != nil {
			log.Fatalf(ErrorMsg, err)
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
		Name:    "StackAuditor",
		Version: pluginVersion,
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
					Options: map[string]string{
						"-csv":  fmt.Sprintf("output results in csv format"),
						"-json": fmt.Sprintf("output results in json format"),
					},
					Usage: AuditStackUsage,
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
					Options: map[string]string{
						"-v3": fmt.Sprintf("Attempts to change stack with zero downtime (EXPERIMENTAL: This requires a minimum CAPI version of %s)", changer.V3ZDTCCAPIMinimum),
					},
					Usage: ChangeStackUsage,
				},
			},
		},
	}
}
