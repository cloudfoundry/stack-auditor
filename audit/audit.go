package audit

import (
	"encoding/json"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/stack-auditor/resources"
)

type Actor struct {
	Conn plugin.CliConnection
}

func (a *Actor) GetOrgs() (resources.Orgs, error) {
	return a.Conn.GetOrgs()
}

func (a *Actor) GetAllSpaces() (resources.Spaces, error) {
	var allSpaces resources.Spaces
	nextSpaceURL := "/v2/spaces"
	for nextSpaceURL != "" {
		spacesJSON, err := a.Conn.CliCommandWithoutTerminalOutput("curl", nextSpaceURL)
		if err != nil {
			return nil, err
		}

		var spaces resources.SpacesJSON
		if err := json.Unmarshal([]byte(strings.Join(spacesJSON, "")), &spaces); err != nil {
			return nil, fmt.Errorf("error unmarshaling spaces json: %v", err)
		}
		nextSpaceURL = spaces.NextURL
		allSpaces = append(allSpaces, spaces)
	}

	return allSpaces, nil
}

func (a *Actor) GetAllStacks() (resources.Stacks, error) {
	var allStacks resources.Stacks
	nextStackURL := "/v2/stacks"
	for nextStackURL != "" {
		stacksJSON, err := a.Conn.CliCommandWithoutTerminalOutput("curl", nextStackURL)
		if err != nil {
			return nil, err
		}

		var stacks resources.StacksJSON
		if err := json.Unmarshal([]byte(strings.Join(stacksJSON, "")), &stacks); err != nil {
			return nil, fmt.Errorf("error unmarshaling stacks json: %v", err)
		}
		nextStackURL = stacks.NextURL
		allStacks = append(allStacks, stacks)
	}

	return allStacks, nil
}

func (a *Actor) GetAllApps() (resources.Apps, error) {
	var allApps resources.Apps
	nextURL := "/v2/apps"
	for nextURL != "" {
		appJSON, err := a.Conn.CliCommandWithoutTerminalOutput("curl", nextURL)
		if err != nil {
			return nil, err
		}

		var apps resources.AppsJSON

		if err := json.Unmarshal([]byte(strings.Join(appJSON, "")), &apps); err != nil {
			return nil, fmt.Errorf("error unmarshaling apps json: %v", err)
		}
		nextURL = apps.NextURL
		allApps = append(allApps, apps)
	}
	return allApps, nil
}
