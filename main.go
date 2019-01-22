package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/cloudfoundry/stack-auditor/changer"

	"github.com/cloudfoundry/stack-auditor/auditor"

	"code.cloudfoundry.org/cli/plugin"
)

type StackAuditor struct{}

const (
	AuditStackCmd  = "audit-stack"
	ChangeStackCmd = "change-stack"
)

func main() {
	plugin.Start(new(StackAuditor))
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

	case ChangeStackCmd:
		if len(args) != 3 {
			log.Fatal("Incorrect number of arguments provided - Usage: cf change-stack <app> <stack>\n")
		}

		c := changer.Changer{
			CF: cf.CF{
				Conn: cliConnection,
			},
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
				HelpText: "Audit stack command's help text",

				UsageDetails: plugin.Usage{
					Usage: AuditStackCmd + "\n   cf audit-stack",
				},
			},
			{
				Name:     ChangeStackCmd,
				HelpText: "Change stack command's help text",

				UsageDetails: plugin.Usage{
					Usage: ChangeStackCmd + "\n   cf change-stack /org/space/app stack",
				},
			},
		},
	}
}
