package cf

import (
	"code.cloudfoundry.org/cli/cf/errors"
	"encoding/json"
	"fmt"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/stack-auditor/resources"
)

type CF struct {
	Conn plugin.CliConnection
}

const (
	V2ResultsPerPage = 100
	V3ResultsPerPage = 5000
)

func (cf *CF) GetAppsAndStacks() ([]string, error) {
	var entries []string

	orgMap, spaceNameMap, spaceOrgMap, allApps, err := cf.getCFContext()
	if err != nil {
		return nil, err
	}

	for _, appsJSON := range allApps {
		for _, app := range appsJSON.Apps {
			appName := app.Name
			spaceName := spaceNameMap[app.Relationships.Space.Data.GUID]
			stackName := app.Lifecycle.Data.Stack

			orgName := orgMap[spaceOrgMap[app.Relationships.Space.Data.GUID]]
			entries = append(entries, fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, appName, stackName))
		}
	}
	return entries, nil
}

func (cf *CF) GetStackGUID(stackName string) (string, error) {
	out, err := cf.Conn.CliCommandWithoutTerminalOutput("stack", "--guid", stackName)
	if err != nil {
		return "", fmt.Errorf("failed to get GUID of %s", stackName)
	}

	if len(out) == 0 {
		return "", fmt.Errorf("%s is not a valid stack", stackName)
	}

	stackGUID := out[0]
	if stackGUID == "" {
		return "", fmt.Errorf("%s is not a valid stack", stackName)
	}

	return stackGUID, nil
}

func (cf *CF) GetAllBuildpacks() ([]resources.BuildpacksJSON, error) {
	var allBuildpacks []resources.BuildpacksJSON
	nextURL := fmt.Sprintf("/v2/buildpacks?results-per-page=%d", V2ResultsPerPage)
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

func (cf *CF) getCFContext() (orgMap, spaceNameMap, spaceOrgMap map[string]string, allApps []resources.V3AppsJSON, err error) {
	orgs, err := cf.getOrgs()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	allSpaces, err := cf.getAllSpaces()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	allApps, err = cf.GetAllApps()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	orgMap = orgs.Map()
	spaceNameMap, spaceOrgMap = allSpaces.MakeSpaceOrgAndNameMap()

	return orgMap, spaceNameMap, spaceOrgMap, allApps, nil
}

func (cf *CF) getOrgs() (resources.Orgs, error) {
	return cf.Conn.GetOrgs()
}

func (cf *CF) getAllSpaces() (resources.Spaces, error) {
	var allSpaces resources.Spaces
	nextSpaceURL := fmt.Sprintf("/v2/spaces?results-per-page=%d", V2ResultsPerPage)
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

func (cf *CF) GetAllApps() ([]resources.V3AppsJSON, error) {
	var allApps []resources.V3AppsJSON
	nextURL := fmt.Sprintf("/v3/apps?per_page=%d", V3ResultsPerPage)
	for nextURL != "" {
		appJSON, err := cf.Conn.CliCommandWithoutTerminalOutput("curl", nextURL)
		if err != nil {
			return nil, err
		}

		var apps resources.V3AppsJSON

		if err := json.Unmarshal([]byte(strings.Join(appJSON, "")), &apps); err != nil {
			return nil, fmt.Errorf("error unmarshaling apps json: %v", err)
		}
		nextURL = apps.Pagination.Next.Href
		allApps = append(allApps, apps)
	}
	return allApps, nil
}

func (cf *CF) GetAppInfo(appName, spaceGuid string) (appGuid, appState, appStack string, err error) {

	apps, err := cf.GetAllApps()
	if err != nil {
		return "", "", "", err
	}

	for _, appsJSON := range apps {
		for _, app := range appsJSON.Apps {
			curApp := app.Name
			curSpace := app.Relationships.Space.Data.GUID

			if curSpace == spaceGuid && curApp == appName {
				return app.GUID, app.State, app.Lifecycle.Data.Stack, nil
			}
		}
	}
	return "", "", "", errors.New("application could not be found")

}
