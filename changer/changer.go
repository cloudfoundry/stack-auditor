package changer

import (
	"fmt"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
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

func (c *Changer) ChangeStack(appName string, newStack string) (string, error) {
	fmt.Printf(AttemptingToChangeStackMsg, newStack, appName)

	appInitialInfo, err := c.CF.GetApp(appName, newStack)
	if err != nil {
		return "", err
	}

	if appInitialInfo.Lifecycle.Data.Stack == newStack {
		return "", fmt.Errorf("application is already associated with stack %s", newStack)
	}

	stackGuid, err := c.CF.GetStackGUID(newStack)
	if err != nil {
		return "", err
	}

	appGuid := appInitialInfo.GUID
	if _, err = c.CF.Conn.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid, "-X", "PUT", `-d={"stack_guid":"`+stackGuid+`","state":"STOPPED"}`); err != nil {
		return "", err
	}

	if appInitialInfo.State == "STARTED" {
		if _, err := c.CF.Conn.CliCommand("start", appName); err != nil {
			return "", err
		}
	}

	result := fmt.Sprintf(ChangeStackSuccessMsg, appName, newStack)
	return result, nil
}
