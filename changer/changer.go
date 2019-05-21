package changer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/buger/jsonparser"
	"github.com/pkg/errors"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	AppStackAssociationError   = "application is already associated with stack %s"
	V3ZDTCapiMinimum           = "1.76.3" // The CAPI release version for PAS 2.5.0
	RestoringStateMsg          = "Restoring prior application state: %s"
	RecoveryMsg                = "%s failed to stage on: %s.\nError: %s\nRestaging on existing stack: %s\n"
	ErrorChangingStack         = "problem assigning targetStack"
	ErrorStaging               = "problem staging new droplet"
	ErrorSettingDroplet        = "problem setting droplet"
	ErrorRetrievingAPIVersion  = "problem retrieving cf api version"
	ErrorCheckingZDTSupport    = "problem checking for ZDT support"
	ErrorRestartingApp         = "problem restarting app"
	ErrorZDTNotSupported       = "your CAPI version does not support a zero downtime restart"
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
	Log    func(writer io.Writer, msg string)
	V3Flag bool
}

type Runner interface {
	Run(bin, dir string, quiet bool, args ...string) error
	RunWithOutput(bin, dir string, quiet bool, args ...string) (string, error)
	SetEnv(variableName string, path string) error
}

func (c *Changer) ChangeStack(appName, newStack string) (string, error) {
	fmt.Printf(AttemptingToChangeStackMsg, newStack, fmt.Sprintf("%s/%s/", c.CF.Space.Name, appName))
	appGuid, appState, appStack, err := c.CF.GetAppInfo(appName)
	if err != nil {
		return "", err
	}

	if appStack == newStack {
		return "", fmt.Errorf(AppStackAssociationError, newStack)
	}

	if err := c.change(appName, appGuid, newStack, appState); err != nil {
		c.Log(os.Stderr, fmt.Sprintf(RecoveryMsg, appName, newStack, err, appStack))
		if err := c.recoverApp(appName, appGuid, appState, appStack); err != nil {
			c.Log(os.Stderr, fmt.Sprintf("unable to recover %s", appName))
			return "", err
		}

		return "", nil
	}

	return fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack), nil
}

func (c *Changer) change(appName, appGUID, stackName, appInitialState string) error {
	version, err := c.GetAPIVersion()
	if err != nil {
		return errors.Wrap(err, ErrorRetrievingAPIVersion)
	}

	zdtExists, err := IsZDTSupported(version)
	if err != nil {
		return errors.Wrap(err, ErrorCheckingZDTSupport)
	}

	if !zdtExists && c.V3Flag {
		return fmt.Errorf(ErrorZDTNotSupported)
	}

	err = c.assignTargetStack(appGUID, stackName)
	if err != nil {
		return errors.Wrap(err, ErrorChangingStack)
	}

	newDropletGUID, err := c.v3Stage(appGUID)
	if err != nil {
		return errors.Wrap(err, ErrorStaging)
	}

	if err := c.v3SetDroplet(appGUID, newDropletGUID); err != nil {
		return errors.Wrap(err, ErrorSettingDroplet)
	}

	if zdtExists && c.V3Flag {
		err = c.restartZDT(appName)
	} else {
		err = c.restartNonZDT(appName, appGUID)
	}

	if err != nil {
		return errors.Wrap(err, ErrorRestartingApp)
	}

	return c.restoreAppState(appGUID, appInitialState)
}

func (c *Changer) assignTargetStack(appGuid, stackName string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"} } }`)
	return err
}

func (c *Changer) GetAPIVersion() (string, error) {
	return c.CF.Conn.ApiVersion()
}

func (c *Changer) restartZDT(appName string) error {
	fmt.Printf("Restarting %s with zero down time...\n", appName)
	return c.Runner.Run("cf", ".", true, "v3-zdt-restart", appName)
}

func (c *Changer) restartNonZDT(appName, appGuid string) error {
	fmt.Printf("Restarting %s with down time...\n", appName)
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid+"/actions/restart", "-X", "POST")
	return err
}

func (c *Changer) v3Stage(appGuid string) (string, error) {
	curDropletResp, err := c.CF.CFCurl("/v3/apps/" + appGuid + "/droplets/current")
	if err != nil {
		return "", err
	}

	packageGUID, err := parsePackageFromDroplet(curDropletResp)
	if err != nil {
		return "", err
	}

	buildPostResp, err := c.CF.CFCurl("/v3/builds", "-X", "POST", `-d='{"package": {"guid": "`+packageGUID+`"} }'`)
	if err != nil {
		return "", err
	}

	buildGUID, err := parseBuildGUID(buildPostResp)
	if err != nil {
		return "", err
	}

	buildGetResp, err := c.waitOnAppBuild(buildGUID)
	if err != nil {
		return "", err
	}

	return parseNewStackDropletGUID(buildGetResp)
}

func (c *Changer) v3SetDroplet(appGUID, dropletGUID string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGUID+"/relationships/current_droplet", "-X", "PATCH", `-d='{ "data": { "guid": "`+dropletGUID+`" } }'`)
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
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid+"/actions/"+action, "-X", "POST")
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
		time.Sleep(5 * time.Second)
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

func IsZDTSupported(version string) (bool, error) {
	limitSemver, _ := semver.Parse(V3ZDTCapiMinimum)
	CAPISemver, err := semver.Parse(version)
	if err != nil {
		return false, err
	}
	return CAPISemver.GTE(limitSemver), nil
}

func (c *Changer) recoverApp(appName, appGuid, appInitialState, appInitialStack string) error {
	version, err := c.GetAPIVersion()
	if err != nil {
		return err
	}

	zdtExists, err := IsZDTSupported(version)
	if err != nil {
		return err
	}

	if zdtExists {
		return c.assignTargetStack(appGuid, appInitialStack)
	}

	return c.change(appName, appGuid, appInitialStack, appInitialState)
}
