package changer

import (
	"fmt"
	"io"
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
	V3ZDTCCAPIMinimum          = "2.131.0" // This is cc-api version from capi-release v1.76.0, which ships with PAS 2.5
	RestoringStateMsg          = "Restoring prior application state: %s"
	ErrorChangingStack         = "problem assigning target stack to %s"
	ErrorStaging               = "problem staging new droplet on %s"
	ErrorSettingDroplet        = "problem setting droplet on %s"
	ErrorRestartingApp         = "problem restarting app on %s"
	ErrorRetrievingAPIVersion  = "problem retrieving cf api version"
	ErrorCheckingZDTSupport           = "problem checking for ZDT support"
	ErrorRecoveringFromStaging        = "Problem recovering from staging error"
	ErrorRecoveringFromRestart        = "Problem recovering from restart error"
	ErrorRecoveringFromSettingDroplet = "Problem recovering from setting the droplet error"
	ErrorZDTNotSupported              = "Your CAPI version does not support a zero downtime restart. Please remove --v3 flag and try again"
	RestagingMsg                      = "Restaging on existing stack: %s\n"
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
	appGuid, appState, oldStack, err := c.CF.GetAppInfo(appName)
	if err != nil {
		return "", err
	}

	if oldStack == newStack {
		return "", fmt.Errorf(AppStackAssociationError, newStack)
	}

	if err := c.change(appName, appGuid, oldStack, newStack, appState); err != nil {
		return "", err
	}

	return fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack), nil
}

func (c *Changer) change(appName, appGUID, oldStack, newStack, appInitialState string) error {
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

	err = c.assignTargetStack(appGUID, newStack)
	if err != nil {
		return errors.Wrapf(err, ErrorChangingStack, newStack)
	}

	newDropletGUID, oldDropletGUID, packageGUID, err := c.v3Stage(appGUID)
	if err != nil {
		err = errors.Wrapf(err, ErrorStaging, newStack)
		if stagingErr := c.recoverStaging(appGUID, oldStack, packageGUID); stagingErr != nil {
			err = errors.Wrap(err, ErrorRecoveringFromStaging)
		}
		return err
	}

	if err := c.v3SetDroplet(appGUID, newDropletGUID); err != nil {
		err = errors.Wrapf(err, ErrorSettingDroplet, newStack)
		if stagingErr := c.recoverStaging(appGUID, oldStack, packageGUID); stagingErr != nil {
			err = errors.Wrap(err, ErrorRecoveringFromSettingDroplet)
		}
		return err
	}

	if zdtExists && c.V3Flag {
		err = c.restartZDT(appName)
	} else {
		err = c.restartNonZDT(appName, appGUID)
	}

	if err != nil {
		err = errors.Wrapf(err, ErrorRestartingApp, newStack)
		if restartErr := c.recoverRestart(appName, appGUID, oldStack, packageGUID, oldDropletGUID); restartErr != nil {
			err = errors.Wrap(err, ErrorRecoveringFromRestart)
		}
		return err
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

func (c *Changer) v3Stage(appGuid string) (string, string, string, error) {
	curDropletResp, err := c.CF.CFCurl("/v3/apps/" + appGuid + "/droplets/current")
	if err != nil {
		return "", "", "", err
	}

	oldDropletGUID, err := parseBuildGUID(curDropletResp)
	if err != nil {
		return "", "", "", err
	}

	packageGUID, err := parsePackageFromDroplet(curDropletResp)
	if err != nil {
		return "", oldDropletGUID, "", err
	}

	buildPostResp, err := c.CF.CFCurl("/v3/builds", "-X", "POST", `-d='{"package": {"guid": "`+packageGUID+`"} }'`)
	if err != nil {
		return "", oldDropletGUID, packageGUID, err
	}

	buildGUID, err := parseBuildGUID(buildPostResp)
	if err != nil {
		return "", oldDropletGUID, packageGUID, err
	}

	buildGetResp, err := c.waitOnAppBuild(buildGUID)
	if err != nil {
		return "", oldDropletGUID, packageGUID, err
	}

	dropletGUID, err := parseNewStackDropletGUID(buildGetResp)

	return dropletGUID, oldDropletGUID, packageGUID, err
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

func (c *Changer) recoverTargetStack(appGUID, oldStack string) error {
	return c.assignTargetStack(appGUID, oldStack)
}

func (c *Changer) recoverStaging(appGUID, oldStack, packageGUID string) error {
	err := c.recoverTargetStack(appGUID, oldStack)
	if err != nil {
		return err
	}

	_, err = c.CF.CFCurl("/v3/builds", "-X", "POST", `-d='{"package": {"guid": "`+packageGUID+`"} }'`)

	return err
}

func (c *Changer) recoverSettingDroplet(appGUID, oldStack, packageGUID, oldDropletGUID string) error {
	if err := c.recoverStaging(appGUID, oldStack, packageGUID); err != nil {
		return err
	}

	if err := c.v3SetDroplet(appGUID, oldDropletGUID); err != nil {
		return err
	}

	return nil
}

func (c *Changer) recoverRestart(appName, appGUID, oldStack, packageGUID, oldDropletGUID string) error {
	if err := c.recoverSettingDroplet(appGUID, oldStack, packageGUID, oldDropletGUID); err != nil {
		return err
	}

	fmt.Printf(RestagingMsg, oldStack)

	var err error
	if c.V3Flag {
		err = c.restartZDT(appName)
	} else {
		err = c.restartNonZDT(appName, appGUID)
	}

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

func parseNewStackDropletGUID(buildGetResp []string) (string, error) {
	dropletGUID, err := jsonparser.GetString([]byte(strings.Join(buildGetResp, "\n")), "droplet", "guid")
	if err != nil {
		return "", err
	}

	return dropletGUID, nil
}

func IsZDTSupported(version string) (bool, error) {
	limitSemver, _ := semver.Parse(V3ZDTCCAPIMinimum)
	CAPISemver, err := semver.Parse(version)
	if err != nil {
		return false, err
	}
	return CAPISemver.GTE(limitSemver), nil
}
