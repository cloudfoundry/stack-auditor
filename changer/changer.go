package changer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/blang/semver"
	"github.com/buger/jsonparser"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	ChangeStackV3ErrorMsg      = "the --v3 flag is not compatible with your foundation. Please remove the flag and rerun"
	AppStackAssociationError   = "application is already associated with stack %s"
	V3ZDTCapiMinimum           = "1.76.3" // The CAPI release version for PAS 2.5.0
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

	if err := c.changeStack(appName, appGuid, newStack, appState); err != nil {
		fmt.Fprintf(os.Stderr, RecoveryMsg, appName, newStack, appStack)

		if err := c.recoverApp(appName, appGuid, appState, appStack); err != nil {
			fmt.Fprintf(os.Stderr, "unable to recover %s", appName)
			return "", err
		}

		return "", nil
	}

	return fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack), nil
}

func (c *Changer) changeStack(appName, appGuid, stackName, appInitialState string) error {
	err := c.assignTargetStack(appGuid, stackName)
	if err != nil {
		return err
	}

	zdtExists, err := c.isZDTSupported()
	if err != nil {
		return err
	}

	if zdtExists {
		return c.restartZDT(appName)
	} else {
		err = c.rebuildApp(appGuid)
		if err != nil {
			return err
		}

	}

	return c.restoreAppState(appGuid, appInitialState)
}

func (c *Changer) assignTargetStack(appGuid, stackName string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"} } }`)
	return err
}

func (c *Changer) isZDTSupported() (bool, error) {
	CAPIZDTLimitSemver, _ := semver.Parse(V3ZDTCapiMinimum)
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

func (c *Changer) restartZDT(appName string) error {
	fmt.Printf("Restarting %s with zero down time...\n", appName)
	return c.Runner.Run("cf", ".", true, "v3-zdt-restart", appName)
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

func (c *Changer) restoreAppState(appGuid, appInitialState string) error {
	var action string

	switch appInitialState {
	case "STARTED":
		action = "start"
	case "STOPPED":
		action = "stop"
	default:
		return fmt.Errorf("unhandled initial application state (%s)", appInitialState)
	}

	fmt.Println(fmt.Sprintf(RestoringStateMsg, appInitialState))
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid+"/actions/" + action, "-X", "POST")
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

	for buildState == "STAGING" {
		buildGetResp, err = c.CF.CFCurl("/v3/builds/" + buildGUID)
		if err != nil {
			return []string{}, err
		}

		buildJSON := strings.Join(buildGetResp, "\n")
		buildState, err = jsonparser.GetString([]byte(buildJSON), "state")
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

func (c *Changer) recoverApp(appName, appGuid, appInitialState, appInitialStack string) error {
	if err := c.changeStack(appName, appGuid, appInitialStack, appInitialState); err != nil {
		return err
	}

	return nil
}
