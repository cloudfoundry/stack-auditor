package changer

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/cloudfoundry/stack-auditor/resources"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	ChangeStackV3ErrorMsg      = "the --v3 flag is not compatible with your foundation. Please remove the flag and rerun"
	AppStackAssociationError   = "application is already associated with stack %s"
)

type RequestData struct {
	LifeCycle struct {
		Data struct {
			Stack string `json:"stack"`
		} `json:"data"`
	} `json:"lifecycle"`
}

type Changer struct {
	CF cf.CF
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
		return "", err
	}

	return fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack), nil
}

func (c *Changer) changeStack(appGuid, stackName, appState string) error {
	response, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"}}}`)
	if err != nil {
		return err
	}

	var app resources.V3App
	if err := json.Unmarshal([]byte(strings.Join(response, "\n")), &app); err != nil {
		return err
	}

	var packageUrl *url.URL
	packageUrl, err = url.Parse(app.Links.Packages.Href)
	if err != nil {
		return err
	}

	//TODO: re write this with static v3 endpoint: /v3/apps/"+appGuid+"/packages
	packagesPathForCf := strings.TrimPrefix(packageUrl.String(), fmt.Sprintf("%s://%s", packageUrl.Scheme, packageUrl.Host))

	packageResponse, err := c.CF.CFCurl(packagesPathForCf, "-X", "GET")
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

	cmd := exec.Command("cf", "v3-stage", app.Name, "--package-guid", packagerJSON.Resources[0].GUID)
	fmt.Printf("Staging %s...\n", app.Name)
	if err := cmd.Run(); err != nil {
		return err
	}

	var dropletJSON resources.DropletJSON

	dropletResponse, err := c.CF.CFCurl("/v3/apps/"+appGuid+"/droplets/current", "-X", "GET")
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(strings.Join(dropletResponse, "\n")), &dropletJSON); err != nil {
		return err
	}

	fmt.Println()
	cmd = exec.Command("cf", "v3-set-droplet", app.Name, "--droplet-guid", dropletJSON.GUID)
	fmt.Printf("Setting droplet for %s...\n", app.Name)
	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR HERE: ", err.Error())
		return err
	}

	return c.restart(app.Name)
}

func (c *Changer) restart(appName string) error {
	if !supportV3ZeroDowntime() {
		// cf restart test-app ---> TODO this is not zero downtime
		return nil
	}

	cmd := exec.Command("cf", "v3-zdt-restart", appName)
	fmt.Printf("Restarting %s...\n", appName)
	if err := cmd.Run(); err != nil {
		fmt.Println("ERROR HERE???????????: ", err.Error())
		return err
	}

	return nil
}

func supportV3ZeroDowntime() bool {
	return true
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
