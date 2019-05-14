package changer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/buger/jsonparser"
	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/cloudfoundry/stack-auditor/resources"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	ChangeStackV3ErrorMsg      = "the --v3 flag is not compatible with your foundation. Please remove the flag and rerun"
	AppStackAssociationError   = "application is already associated with stack %s"
	V3ZDTCapiLimit             = "1.76.3"
	RestoringStateMsg          = "Restoring prior application state: %s"
	RecoveryMsg                = "%s failed to stage on: %s. Restaging on existing stack: %s\n"
)

type RequestData struct {
	LifeCycle struct {
		Data struct {
			Stack string `json:"stack"`
		} `json:"data"`
	} `json:"lifecycle"`
}

type Changer struct {
	CF     cf.CF
	Runner Runner
}

type Runner interface {
	Run(bin, dir string, quiet bool, args ...string) error
	RunWithOutput(bin, dir string, quiet bool, args ...string) (string, error)
	SetEnv(variableName string, path string) error
}

func (c *Changer) ChangeStack(appName, newStack string, v3Flag bool) (string, error) {
	fmt.Printf(AttemptingToChangeStackMsg, newStack, appName)
	appGuid, appState, appStack, err := c.CF.GetAppInfo(appName)
	if err != nil {
		return "", err
	}

	if appStack == newStack {
		return "", fmt.Errorf(AppStackAssociationError, newStack)
	}

	if err := c.changeStack(appGuid, newStack, appState); err != nil {
		fmt.Fprintf(os.Stderr, RecoveryMsg, appName, newStack, appStack)

		if err := c.recover(appGuid, appState, appStack); err != nil {
			fmt.Fprintf(os.Stderr, "unable to recover %s", appName)
			return "", err
		}

		return "", nil
	}

	return fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack), nil
}

func (c *Changer) changeStack(appGuid, stackName, appInitialState string) error {
	err := c.assignDesiredStack(appGuid, stackName)
	if err != nil {
		return err
	}

	err = c.rebuildApp(appGuid)
	if err != nil {
		return err
	}

	return c.restoreAppState(appGuid, appInitialState)
}

func (c *Changer) restoreAppState(appGuid, appInitialState string) error {
	if appInitialState == "STARTED" {
		fmt.Println(fmt.Sprintf(RestoringStateMsg, "STARTED"))
		_, err := c.CF.CFCurl("/v3/apps/"+appGuid+"/actions/restart", "-X", "POST")
		if err != nil {
			return err
		}
	} else {
		fmt.Println(fmt.Sprintf(RestoringStateMsg, "STOPPED"))
	}

	return nil
}

func (c *Changer) assignDesiredStack(appGuid, stackName string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"} } }`)
	return err
}

func (c *Changer) rebuildApp(appGuid string) error {
	curDropletResp, err := c.CF.CFCurl("/v3/apps/" + appGuid + "/droplets/current")
	if err != nil {
		return err
	}

	packageGUID, err := parsePackageFromDroplet(curDropletResp)
	if err != nil {
		return err
	}

	buildPostResp, err := c.CF.CFCurl("/v3/builds", "-X", "POST", `-d='{"package": {"guid": "`+packageGUID+`"} }'`)
	if err != nil {
		return err
	}

	buildGUID, err := parseBuildGUID(buildPostResp)
	if err != nil {
		return err
	}

	buildGetResp, err := c.waitOnAppBuild(buildGUID)
	if err != nil {
		return err
	}

	newStackDropletGUID, err := parseNewStackDropletGUID(buildGetResp)
	_, err = c.CF.CFCurl("/v3/apps/"+appGuid+"/relationships/current_droplet", "-X", "PATCH", `-d='{ "data": { "guid": "`+newStackDropletGUID+`" } }'`)
	return err
}

func parsePackageFromDroplet(curDropletResp []string) (string, error) {
	packageURI, err := jsonparser.GetString([]byte(strings.Join(curDropletResp, "\n")), "links", "package", "href")
	if err != nil {
		return "", err
	}

	return filepath.Base(packageURI), nil
}

func parseBuildGUID(buildPostResp []string) (string, error) {
	buildGUID, err := jsonparser.GetString([]byte(strings.Join(buildPostResp, "\n")), "guid")
	if err != nil {
		return "", err
	}

	return buildGUID, nil
}

func (c *Changer) waitOnAppBuild(buildGUID string) (buildGetResp []string, err error) {
	buildState := "STAGING"
	deadline := time.Now().Add(3 * time.Hour)

	for buildState == "STAGING" && time.Now().Before(deadline) {
		buildGetResp, err = c.CF.CFCurl("/v3/builds/" + buildGUID)
		if err != nil {
			return []string{}, err
		}

		buildJSON := strings.Join(buildGetResp, "\n")
		buildState, err = jsonparser.GetString([]byte(buildJSON), "state")
	}

	if time.Now().After(deadline) {
		return []string{}, fmt.Errorf("timed out waiting for app to build. build GUID: %s", buildGUID)
	}

	if buildState == "FAILED" {
		return []string{}, fmt.Errorf("app build failed. build GUID: %s", buildGUID)
	}

	return buildGetResp, nil
}

func parseNewStackDropletGUID(buildGetResp []string) (string, error) {
	dropletGUID, err := jsonparser.GetString([]byte(strings.Join(buildGetResp, "\n")), "droplet", "guid")
	if err != nil {
		return "", err
	}

	return dropletGUID, nil
}

// TODO remove this once finished
func (c *Changer) oldChangeStack(appGuid, stackName, appState string) error {
	response, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"} } }`)
	if err != nil {
		return err
	}

	var app resources.V3App
	if err := json.Unmarshal([]byte(strings.Join(response, "\n")), &app); err != nil {
		return err
	}

	packageResponse, err := c.CF.CFCurl(fmt.Sprintf("/v3/apps/%s/packages", appGuid), "-X", "GET")
	if err != nil {
		return err
	}

	var packagerJSON resources.PackagerJSON
	if err := json.Unmarshal([]byte(strings.Join(packageResponse, "\n")), &packagerJSON); err != nil {
		return err
	}

	if len(packagerJSON.Resources) < 1 {
		return fmt.Errorf("error parsing packager GUID from json, there are no packages")
	}

	fmt.Println("Packager GUID: ", packagerJSON.Resources[0].GUID)

	fmt.Printf("Staging %s...\n", app.Name)
	if err := c.Runner.Run("cf", ".", false, "v3-stage", app.Name, "--package-guid", packagerJSON.Resources[0].GUID); err != nil {
		return err
	}

	var dropletsJSON resources.DropletListJSON

	dropletResponse, err := c.CF.CFCurl("/v3/apps/"+appGuid+"/droplets", "-X", "GET")
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(strings.Join(dropletResponse, "\n")), &dropletsJSON); err != nil {
		return err
	}

	dropletGuid := dropletsJSON.Resources[len(dropletsJSON.Resources)-1].GUID
	fmt.Printf("Setting droplet for %s to %s...\n", app.Name, dropletGuid)
	if err := c.Runner.Run("cf", ".", false, "v3-set-droplet", app.Name, "--droplet-guid", dropletGuid); err != nil {
		return err
	}

	return c.restart(app.Name)
}

// TODO update this
func (c *Changer) restart(appName string) error {
	ok, err := c.supportV3ZeroDowntime()
	if err != nil {
		return err
	}

	if !ok {
		fmt.Printf("Restarting %s...\n", appName)
		if err := c.Runner.Run("cf", ".", false, "restart", appName); err != nil {
			return err
		}
		return nil
	}

	fmt.Printf("Restarting %s with zero down time...\n", appName)
	if err := c.Runner.Run("cf", ".", false, "v3-zdt-restart", appName); err != nil {
		return err
	}

	return nil
}

func (c *Changer) supportV3ZeroDowntime() (bool, error) {
	CAPIZDTLimitSemver, _ := semver.Parse(V3ZDTCapiLimit)
	currentCAPIVersion, err := c.CF.Conn.ApiVersion()
	if err != nil {
		return false, err
	}
	currentCAPISemver, err := semver.Parse(currentCAPIVersion)
	if err != nil {
		return false, err
	}

	return currentCAPISemver.GTE(CAPIZDTLimitSemver), nil
}

func (c *Changer) changeStackV3(appGuid, stackName string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"}}}`)
	if err != nil {
		return err
	}

	return nil
}

func (c *Changer) changeStackV2(appName, appGuid, newStackGuid, appState string) error {
	_, err := c.CF.CFCurl("/v2/apps/"+appGuid, "-X", "PUT", `-d={"stack_guid":"`+newStackGuid+`","state":"STOPPED"}`)
	if err != nil {
		return err
	}
	if appState == "STARTED" {
		if _, err := c.CF.Conn.CliCommand("start", appName); err != nil {
			return err
		}
	}
	return nil
}

func (c *Changer) recover(appGuid, appInitialState, appStack string) error {
	if err := c.changeStack(appGuid, appStack, appInitialState); err != nil {
		return err
	}

	return nil
}
