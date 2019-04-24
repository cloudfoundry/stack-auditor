package changer

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry/stack-auditor/utils"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
	ChangeStackV3ErrorMsg      = "the --v3 flag is not compatible with your foundation. Please remove the flag and rerun"
	CFNotFound                 = "CF-NotFound"
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

	curSpace, err := c.CF.Conn.GetCurrentSpace()
	if err != nil {
		return "", err
	}

	appGuid, appState, appStack, err := c.CF.GetAppInfo(appName, curSpace.Guid)
	if err != nil {
		return "", err
	}

	if appStack == newStack {
		return "", fmt.Errorf("application is already associated with stack %s", newStack)
	}

	stackGuid, err := c.CF.GetStackGUID(newStack)
	if err != nil {
		return "", err
	}

	if v3Flag {
		if err := c.changeStackV3(appGuid, newStack); err != nil {
			return "", err
		}

	} else {
		if err := c.changeStackV2(appName, appGuid, stackGuid, appState); err != nil {
			return "", err
		}
	}

	result := fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack)
	return result, nil
}

func (c *Changer) changeStackV3(appGuid, stackName string) error {
	out, err := c.CF.Conn.CliCommandWithoutTerminalOutput("curl", "/v3/apps/"+appGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+stackName+`"}}}`)
	if err != nil {
		return err
	}

	notFoundError := utils.CheckOutputForErrorMessage(strings.Join(out, ""), CFNotFound)
	if notFoundError {
		return fmt.Errorf(ChangeStackV3ErrorMsg)
	}

	return nil
}

func (c *Changer) changeStackV2(appName, appGuid, newStackGuid, appState string) error {
	_, err := c.CF.Conn.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid, "-X", "PUT", `-d={"stack_guid":"`+newStackGuid+`","state":"STOPPED"}`)
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
