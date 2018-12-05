package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/cloudfoundry/stack-auditor/structsJSON"
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

func Audit(cliConnection plugin.CliConnection) (string, error) {
	orgs, err := cliConnection.GetOrgs()
	if err != nil {
		return "", err
	}
	orgMap := makeOrgMap(orgs)

	allSpaces, err := getAllSpaces(cliConnection)
	if err != nil {
		return "", err
	}
	spaceNameMap, spaceOrgMap := makeSpaceOrgAndNameMap(allSpaces)

	allStacks, err := getAllStacks(cliConnection)
	if err != nil {
		return "", err
	}
	stackMap := makeStackMap(allStacks)

	allApps, err := getAllApps(cliConnection)
	if err != nil {
		return "", err
	}

	list := assembleEntries(orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps)
	sort.Strings(list)

	return strings.Join(list, "\n") + "\n", nil
}

func getAllSpaces(cliConnection plugin.CliConnection) ([]structsJSON.Spaces, error) {
	var allSpaces []structsJSON.Spaces
	nextSpaceURL := "/v2/spaces"
	for nextSpaceURL != "" {
		spacesJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", nextSpaceURL)
		if err != nil {
			return nil, err
		}

		var spaces structsJSON.Spaces
		if err := json.Unmarshal([]byte(strings.Join(spacesJSON, "")), &spaces); err != nil {
			return nil, fmt.Errorf("error unmarshaling spaces json: %v", err)
		}
		nextSpaceURL = spaces.NextURL
		allSpaces = append(allSpaces, spaces)
	}

	return allSpaces, nil
}

func getAllStacks(cliConnection plugin.CliConnection) ([]structsJSON.Stacks, error) {
	var allStacks []structsJSON.Stacks
	nextStackURL := "/v2/stacks"
	for nextStackURL != "" {
		stacksJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", nextStackURL)
		if err != nil {
			return nil, err
		}

		var stacks structsJSON.Stacks
		if err := json.Unmarshal([]byte(strings.Join(stacksJSON, "")), &stacks); err != nil {
			return nil, fmt.Errorf("error unmarshaling stacks json: %v", err)
		}
		nextStackURL = stacks.NextURL
		allStacks = append(allStacks, stacks)
	}

	return allStacks, nil
}

func getAllApps(cliConnection plugin.CliConnection) ([]structsJSON.Apps, error) {
	var allApps []structsJSON.Apps
	nextURL := "/v2/apps"
	for nextURL != "" {
		appJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", nextURL)
		if err != nil {
			return nil, err
		}

		var apps structsJSON.Apps

		if err := json.Unmarshal([]byte(strings.Join(appJSON, "")), &apps); err != nil {
			return nil, fmt.Errorf("error unmarshaling apps json: %v", err)
		}
		nextURL = apps.NextURL
		allApps = append(allApps, apps)
	}
	return allApps, nil
}

func makeSpaceOrgAndNameMap(allSpaces []structsJSON.Spaces) (map[string]string, map[string]string) {
	spaceOrgMap := make(map[string]string)
	spaceNameMap := make(map[string]string)
	for _, spaces := range allSpaces {
		for _, space := range spaces.Resources {
			spaceNameMap[space.Metadata.GUID] = space.Entity.Name
			spaceOrgMap[space.Metadata.GUID] = space.Entity.OrganizationGUID
		}
	}
	return spaceNameMap, spaceOrgMap
}

func makeStackMap(allStacks []structsJSON.Stacks) map[string]string {
	stackMap := make(map[string]string)
	for _, stacks := range allStacks {
		for _, stack := range stacks.Resources {
			stackMap[stack.Metadata.GUID] = stack.Entity.Name
		}
	}
	return stackMap
}

func makeOrgMap(orgs []plugin_models.GetOrgs_Model) map[string]string {
	m := make(map[string]string)

	for _, org := range orgs {
		m[org.Guid] = org.Name
	}
	return m
}

func assembleEntries(orgMap, spaceMap, spaceOrgMap, stackMap map[string]string, allApps []structsJSON.Apps) []string {
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
