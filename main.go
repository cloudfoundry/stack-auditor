package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"code.cloudfoundry.org/cli/plugin/models"

	"github.com/cloudfoundry/stack-auditor/structsJSON"

	"code.cloudfoundry.org/cli/plugin"
)

type Plugin struct{}

func main() {
	plugin.Start(new(Plugin))
}

func (c *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
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

func Audit(cliConnection plugin.CliConnection) (string, error) {
	orgs, err := cliConnection.GetOrgs()
	if err != nil {
		return "", err
	}

	orgMap := makeOrgMap(orgs)
	spaceJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/spaces")
	if err != nil {
		return "", err
	}
	stackJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/stacks")
	if err != nil {
		return "", err
	}
	appJSON, err := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/apps")
	if err != nil {
		return "", err
	}

	var spaces structsJSON.Spaces
	var stacks structsJSON.Stacks
	var apps structsJSON.Apps

	if err := json.Unmarshal([]byte(strings.Join(spaceJSON, "")), &spaces); err != nil {
		return "", fmt.Errorf("error unmarshaling spaces json: %v", err)
	}
	if err := json.Unmarshal([]byte(strings.Join(stackJSON, "")), &stacks); err != nil {
		return "", fmt.Errorf("error unmarshaling stacks json: %v", err)
	}
	if err := json.Unmarshal([]byte(strings.Join(appJSON, "")), &apps); err != nil {
		return "", fmt.Errorf("error unmarshaling apps json: %v", err)
	}

	spaceMap := spaces.MakeSpaceNameMap()
	spaceOrgMap := spaces.MakeSpaceOrgMap()
	stackMap := stacks.MakeStackMap()

	list := assembleData(orgMap, spaceMap, spaceOrgMap, stackMap, apps)

	sort.Strings(list)

	return strings.Join(list, "\n") + "\n", nil
}

func assembleData(orgMap, spaceMap, spaceOrgMap, stackMap map[string]string, apps structsJSON.Apps) []string {
	var entries []string
	for _, app := range apps.Resources {
		name := app.Entity.Name
		spaceGUID := app.Entity.SpaceGUID
		stackGUID := app.Entity.StackGUID
		orgName := orgMap[spaceOrgMap[spaceGUID]]
		spaceName := spaceMap[spaceGUID]
		stackName := stackMap[stackGUID]
		entries = append(entries, fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, name, stackName))
	}
	return entries
}

func unmarshalToObj(JSON []string, receiver *interface{}) error {
	if err := json.Unmarshal([]byte(strings.Join(JSON, "")), receiver); err != nil {
		return fmt.Errorf("error unmarshaling spaces json: %v", err)
	}
	return nil
}

func makeOrgMap(orgs []plugin_models.GetOrgs_Model) map[string]string {
	m := make(map[string]string)

	for _, org := range orgs {
		m[org.Guid] = org.Name
	}
	return m
}
