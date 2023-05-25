package changer

import (
	"fmt"
	"io"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	AppStackAssociationError   = "application is already associated with stack %s"
	RestoringStateMsg          = "Restoring prior application state: %s"
	ErrorChangingStack         = "problem assigning target stack to %s"
	ErrorRestagingApp          = "problem restaging app on %s"
	ErrorRestoringState        = "problem restoring application state to %s"
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
	err := c.assignTargetStack(appGUID, newStack)
	if err != nil {
		return fmt.Errorf(ErrorChangingStack+": %w", newStack, err)
	}

	err = c.Runner.Run("cf", ".", true, "restage", "--strategy", "rolling", appName)

	if err != nil {
		err = fmt.Errorf(ErrorRestagingApp+": %w", newStack, err)
		if restartErr := c.assignTargetStack(appGUID, oldStack); restartErr != nil {
			err = fmt.Errorf(ErrorChangingStack+": %w", oldStack, err)
		}
		if restoreErr := c.restoreAppState(appGUID, appInitialState); restoreErr != nil {
			err = fmt.Errorf(ErrorRestoringState+": %w", appInitialState, err)
		}
		return err
	}

	return c.restoreAppState(appGUID, appInitialState)
}

func (c *Changer) assignTargetStack(appGuid, stackName string) error {
	_, err := c.CF.CFCurl("/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"} } }`)
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
