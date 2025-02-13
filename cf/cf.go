package cf

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/cloudfoundry/stack-auditor/resources"
)

type CF struct {
	Conn  plugin.CliConnection
	Space plugin_models.Space
}

var (
	V2ResultsPerPage = "100"
	V3ResultsPerPage = "5000"
)

func (cf *CF) GetAppsAndStacks() (resources.Apps, error) {
	var entries []resources.App

	orgMap, spaceNameMap, spaceOrgMap, allApps, err := cf.getCFContext()
	if err != nil {
		return nil, err
	}

	for _, appsJSON := range allApps {
		for _, app := range appsJSON.Apps {
			appName := app.Name
			spaceName := spaceNameMap[app.Relationships.Space.Data.GUID]
			stackName := app.Lifecycle.Data.Stack
			state := strings.ToLower(app.State)

			orgName := orgMap[spaceOrgMap[app.Relationships.Space.Data.GUID]]
			entries = append(entries, resources.App{
				Space: spaceName,
				Name:  appName,
				Stack: stackName,
				Org:   orgName,
				State: state,
			})
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
	nextURL := fmt.Sprintf("/v2/buildpacks?results-per-page=%s", V2ResultsPerPage)
	for nextURL != "" {
		buildpackJSON, err := cf.CFCurl(nextURL)

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
	nextSpaceURL := fmt.Sprintf("/v2/spaces?results-per-page=%s", V2ResultsPerPage)
	for nextSpaceURL != "" {
		spacesJSON, err := cf.CFCurl(nextSpaceURL)
		if err != nil {
			return nil, err
		}

		var spaces resources.SpacesJSON
		if strings.Join(spacesJSON, "") == "" {
			break
		}
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
	nextURL := fmt.Sprintf("/v3/apps?per_page=%s", V3ResultsPerPage)
	for nextURL != "" {
		appJSON, err := cf.CFCurl(nextURL)
		if err != nil {
			return nil, err
		}

		var apps resources.V3AppsJSON
		if strings.Join(appJSON, "") == "" {
			break
		}

		if err := json.Unmarshal([]byte(strings.Join(appJSON, "")), &apps); err != nil {
			return nil, fmt.Errorf("error unmarshaling apps json: %v", err)
		}
		nextURL = apps.Pagination.Next.Href
		allApps = append(allApps, apps)
	}
	return allApps, nil
}

func (cf *CF) GetAppByName(appName string) (resources.V3App, error) {
	var apps resources.V3AppsJSON
	var app resources.V3App

	endpoint := fmt.Sprintf("/v3/apps?names=%s&space_guids=%s", url.QueryEscape(appName), cf.Space.Guid)
	appJSON, err := cf.CFCurl(endpoint)
	if err != nil {
		return app, err
	}

	if err := json.Unmarshal([]byte(strings.Join(appJSON, "")), &apps); err != nil {
		return app, fmt.Errorf("error unmarshaling apps json: %v", err)
	}
	if len(apps.Apps) == 0 {
		return app, fmt.Errorf("no app found with name %s", appName)
	}

	app = apps.Apps[0]
	return app, nil
}

func (cf *CF) GetAppInfo(appName string) (appGuid, appState, appStack string, err error) {
	app, err := cf.GetAppByName(appName)
	if err != nil {
		return "", "", "", err
	}

	return app.GUID, app.State, app.Lifecycle.Data.Stack, nil
}

func (cf *CF) CFCurl(path string, args ...string) ([]string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	u.Scheme = ""
	u.Host = ""

	curlArgs := []string{"curl", u.String()}
	curlArgs = append(curlArgs, args...)
	output, err := cf.Conn.CliCommandWithoutTerminalOutput(curlArgs...)
	if err != nil {
		return nil, err
	}

	if err := checkV2Error(output); err != nil {
		return nil, err
	}

	if err := checkV3Error(output); err != nil {
		return nil, err
	}

	return output, nil
}

func checkV2Error(lines []string) error {
	output := strings.Join(lines, "\n")
	var errorsJSON resources.V2ErrorJSON

	err := json.Unmarshal([]byte(output), &errorsJSON)

	if err != nil || errorsJSON.Description == "" {
		return nil
	}

	return errors.New(errorsJSON.Description)
}

func checkV3Error(lines []string) error {
	output := strings.Join(lines, "\n")
	var errorsJSON resources.V3ErrorJSON

	err := json.Unmarshal([]byte(output), &errorsJSON)

	if err != nil || errorsJSON.Errors == nil {
		return nil

	}

	errorDetails := make([]string, 0)
	for _, e := range errorsJSON.Errors {
		errorDetails = append(errorDetails, e.Detail)
	}

	return errors.New(strings.Join(errorDetails, ", "))
}
