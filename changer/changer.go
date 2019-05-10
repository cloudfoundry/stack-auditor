package changer

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/buger/jsonparser"

	"github.com/blang/semver"

	"github.com/cloudfoundry/stack-auditor/resources"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	ChangeStackV3ErrorMsg      = "the --v3 flag is not compatible with your foundation. Please remove the flag and rerun"
	AppStackAssociationError   = "application is already associated with stack %s"
	V3ZDTCapiLimit             = "1.76.3"
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

	if err := c.changeStackForRizzle(appGuid, newStack, appState); err != nil {
		return "", err
	}

	return fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack), nil
}

func (c *Changer) changeStackForRizzle(appGuid, stackName, appState string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"} } }`)
	if err != nil {
		return err
	}

	fmt.Println("YARRRRRRRRRRRRR")

	curDropletResp, err := c.CF.CFCurl("/v3/apps/" + appGuid + "/droplets/current")
	if err != nil {
		return err
	}

	packageGUID, err := parsePackageFromDroplet(curDropletResp)
	if err != nil {
		return err
	}

	fmt.Println("YARRRRRRRRRRRRR")

	buildPostResp, err := c.CF.CFCurl("/v3/builds", "-X", "POST", `-d='{"package": {"guid": "`+packageGUID+`"} }'`)
	if err != nil {
		return err
	}

	buildGUID, err := parseBuildGUID(buildPostResp)
	if err != nil {
		return err
	}

	fmt.Println("YARRRRRRRRRRRRR")

	//buildGetResp, err := c.CF.CFCurl("/v3/builds/" + buildGUID)
	//if err != nil {
	//	return err
	//}
	//
	//fmt.Println("YARRRRRRRRRRRRR")
	//fmt.Println("build GET response: ", buildGetResp)
	//
	//newStackDropletGUID, err := parseNewStackDropletGUID(buildGetResp)
	//if err != nil {
	//	return err
	//}

	newStackDropletGUID, err := c.pollDropletBuilding(buildGUID)

	fmt.Println("YARRRRRRRRRRRRR")

	_, err = c.CF.CFCurl("/v3/apps/"+appGuid+"/relationships/current_droplet", "-X", "PATCH", `-d='{ "data": { "guid": "`+newStackDropletGUID+`" } }'`)
	if err != nil {
		return err
	}

	fmt.Println("YARRRRRRRRRRRRR")

	//_, err = c.CF.CFCurl("/v3/apps/"+appGuid+"/actions/restart", "-X", "POST")
	return err
}

func (c *Changer) pollDropletBuilding(buildGUID string) (string, error) {
	dropletGUID := ""
	for dropletGUID == "" {
		buildGetResp, err := c.CF.CFCurl("/v3/builds/" + buildGUID)
		if err != nil {
			return "", err
		}
		dropletGUID, _ = parseNewStackDropletGUID(buildGetResp)
	}
	return dropletGUID, nil
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

func parseNewStackDropletGUID(buildGetResp []string) (string, error) {
	dropletGUID, err := jsonparser.GetString([]byte(strings.Join(buildGetResp, "\n")), "droplet", "guid")
	if err != nil {
		return "", err
	}

	return dropletGUID, nil
}

func (c *Changer) changeStack(appGuid, stackName, appState string) error {
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
