package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/stack-auditor/audit"
	"github.com/cloudfoundry/stack-auditor/resources"
)

type StackAuditor struct{}

func main() {
	plugin.Start(new(StackAuditor))
}

func (s *StackAuditor) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 {
		err := errors.New("no command line arguments provided")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Ensure that we called the command audit-stack
	switch args[0] {
	case "audit-stack":
		info, err := Audit(cliConnection)
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
				Name:     "audit-stack",
				HelpText: "Audit stack command's help text",

				UsageDetails: plugin.Usage{
					Usage: "audit-stack\n   cf audit-stack",
				},
			},
		},
	}
}

func Audit(cli plugin.CliConnection) (string, error) {
	a := audit.Actor{
		Conn: cli,
	}

	orgs, err := a.GetOrgs()
	if err != nil {
		return "", err
	}

	allSpaces, err := a.GetAllSpaces()
	if err != nil {
		return "", err
	}

	allStacks, err := a.GetAllStacks()
	if err != nil {
		return "", err
	}

	allApps, err := a.GetAllApps()
	if err != nil {
		return "", err
	}

	orgMap := orgs.Map()
	spaceNameMap, spaceOrgMap := allSpaces.MakeSpaceOrgAndNameMap()
	stackMap := allStacks.MakeStackMap()

	list := assembleEntries(orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps)
	sort.Strings(list)

	return strings.Join(list, "\n") + "\n", nil
}

func assembleEntries(orgMap, spaceMap, spaceOrgMap, stackMap map[string]string, allApps resources.Apps) []string {
	var entries []string

	for _, apps := range allApps {
		for _, app := range apps.Resources {
			appName := app.Entity.Name
			spaceName := spaceMap[app.Entity.SpaceGUID]
			stackName := stackMap[app.Entity.StackGUID]

			orgName := orgMap[spaceOrgMap[app.Entity.SpaceGUID]]
			entries = append(entries, fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, appName, stackName))
		}
	}
	return entries
}
