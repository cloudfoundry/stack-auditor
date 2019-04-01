package changer

import (
	"fmt"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AttemptingToChangeStackMsg = "Attempting to change stack to %s for %s...\n\n"
	ChangeStackSuccessMsg      = "Application %s was successfully changed to Stack %s"
)

type Changer struct {
	CF cf.CF
}

func (c *Changer) ChangeStack(appName string, stackName string) (string, error) {
	fmt.Printf(AttemptingToChangeStackMsg, stackName, appName)

	stackGuid, err := c.CF.GetStackGUID(stackName)
	if err != nil {
		return "", err
	}

	appInitialInfo, err := c.CF.GetApp(appName, stackName)
	if err != nil {
		return "", err
	}

	if appInitialInfo.Entity.StackGUID == stackGuid {
		return "", fmt.Errorf("application is already associated with stack %s", stackName)
	}

	appGuid := appInitialInfo.Metadata.GUID
	if _, err = c.CF.Conn.CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+appGuid, "-X", "PUT", `-d={"stack_guid":"`+stackGuid+`","state":"STOPPED"}`); err != nil {
		return "", err
	}

	if appInitialInfo.Entity.State == "STARTED" {
		if _, err := c.CF.Conn.CliCommand("start", appName); err != nil {
			return "", err
		}
	}

	result := fmt.Sprintf(ChangeStackSuccessMsg, appName, stackName)
	return result, nil
}
