package cf

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/stack-auditor/resources"
)

type CF struct {
	Conn plugin.CliConnection
}

func (cf *CF) GetAppsAndStacks() ([]string, error) {
	var entries []string

	orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps, err := cf.getCFContext()
	if err != nil {
		return nil, err
	}

	for _, appsJSON := range allApps {
		for _, app := range appsJSON.Apps {
			appName := app.Entity.Name
			spaceName := spaceNameMap[app.Entity.SpaceGUID]
			stackName := stackMap[app.Entity.StackGUID]

			orgName := orgMap[spaceOrgMap[app.Entity.SpaceGUID]]
			entries = append(entries, fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, appName, stackName))
		}
	}
	return entries, nil
}

func (cf *CF) GetStackGUID(stackName string) (string, error) {
	allStacks, err := cf.getAllStacks()
	if err != nil {
		return "", err
	}

	stackGuid := ""
	stackMap := allStacks.MakeStackMap()
	for guid, val := range stackMap {
		if val == stackName {
			stackGuid = guid
			break
		}
	}
	if stackGuid == "" {
		return "", fmt.Errorf("%s is not a valid stack", stackName)
	}

	return stackGuid, nil
}

func (cf *CF) GetApp(appName string, stackName string) (resources.App, error) {
	orgMap, spaceNameMap, spaceOrgMap, _, allApps, err := cf.getCFContext()
	if err != nil {
		return resources.App{}, err
	}

	org, err := cf.Conn.GetCurrentOrg()
	if err != nil {
		return resources.App{}, err
	}

	space, err := cf.Conn.GetCurrentSpace()
	if err != nil {
		return resources.App{}, err
	}

	for _, appsJSON := range allApps {
		for _, cur := range appsJSON.Apps {
			curApp := cur.Entity.Name
			curSpace := spaceNameMap[cur.Entity.SpaceGUID]
			curOrg := orgMap[spaceOrgMap[cur.Entity.SpaceGUID]]

			if curOrg == org.Name && curSpace == space.Name && curApp == appName {
				return cur, nil
			}
		}
	}

	return resources.App{}, errors.New("application could not be found")
}

func (cf *CF) GetAllBuildpacks() ([]resources.BuildpacksJSON, error) {
	var allBuildpacks []resources.BuildpacksJSON
	nextURL := "/v2/buildpacks"
	for nextURL != "" {
		buildpackJSON, err := cf.Conn.CliCommandWithoutTerminalOutput("curl", nextURL)
		if err != nil {
			return nil, err
		}

		var buildpacks resources.BuildpacksJSON

		if err := json.Unmarshal([]byte(strings.Join(buildpackJSON, "")), &buildpacks); err != nil {
			return nil, fmt.Errorf("error unmarshaling apps json: %v", err)
		}
		nextURL = buildpacks.NextURL
		allBuildpacks = append(allBuildpacks, buildpacks)
	}
	return allBuildpacks, nil
}

func (cf *CF) getCFContext() (orgMap, spaceNameMap, spaceOrgMap, stackMap map[string]string, allApps []resources.AppsJSON, err error) {
	orgs, err := cf.getOrgs()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	allSpaces, err := cf.getAllSpaces()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	allStacks, err := cf.getAllStacks()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	allApps, err = cf.GetAllApps()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	orgMap = orgs.Map()
	spaceNameMap, spaceOrgMap = allSpaces.MakeSpaceOrgAndNameMap()
	stackMap = allStacks.MakeStackMap()

	return orgMap, spaceNameMap, spaceOrgMap, stackMap, allApps, nil
}

func (cf *CF) getOrgs() (resources.Orgs, error) {
	return cf.Conn.GetOrgs()
}

func (cf *CF) getAllSpaces() (resources.Spaces, error) {
	var allSpaces resources.Spaces
	nextSpaceURL := "/v2/spaces"
	for nextSpaceURL != "" {
		spacesJSON, err := cf.Conn.CliCommandWithoutTerminalOutput("curl", nextSpaceURL)
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

func (cf *CF) getAllStacks() (resources.Stacks, error) {
	var allStacks resources.Stacks
	nextStackURL := "/v2/stacks"
	for nextStackURL != "" {
		stacksJSON, err := cf.Conn.CliCommandWithoutTerminalOutput("curl", nextStackURL)
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

func (cf *CF) GetAllApps() ([]resources.AppsJSON, error) {
	var allApps []resources.AppsJSON
	nextURL := "/v2/apps"
	for nextURL != "" {
		appJSON, err := cf.Conn.CliCommandWithoutTerminalOutput("curl", nextURL)
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
