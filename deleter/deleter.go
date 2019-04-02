package deleter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	DeleteStackSuccessMsg   = "Stack %s has been deleted"
	DeleteStackBuildpackErr = "you still have buildpacks associated to %s. Please use the `cf delete-buildpack` command to remove associated buildpacks and try again"
	DeleteStackAppErr       = "failed to delete stack %s. You still have apps associated to this stack. Migrate those first."
)

type Deleter struct {
	CF cf.CF
}

func (d *Deleter) DeleteStack(stackName string) (string, error) {
	if err := d.hasAppAssociation(stackName); err != nil {
		return "", err
	}

	if err := d.hasBuildpackAssociation(stackName); err != nil {
		return "", err
	}

	stackGuid, err := d.CF.GetStackGUID(stackName)
	if err != nil {
		return "", err
	}

	lines, err := d.CF.Conn.CliCommandWithoutTerminalOutput("curl", "--fail", "/v2/stacks/"+stackGuid, "-X", "DELETE")
	if err != nil {
		return "", err
	}

	out := strings.Join(lines, "\n")
	if err := checkCurlDelete(out, stackName); err != nil {
		return "", err
	}

	result := fmt.Sprintf(DeleteStackSuccessMsg, stackName)
	return result, nil
}

func (d *Deleter) hasBuildpackAssociation(stackName string) error {
	buildpackMetas, err := d.CF.GetAllBuildpacks()
	if err != nil {
		return err
	}

	for _, buildpackMeta := range buildpackMetas {
		for _, buildpack := range buildpackMeta.BuildPacks {
			if buildpack.Entity.Stack == stackName {
				return fmt.Errorf(DeleteStackBuildpackErr, stackName)
			}
		}
	}

	return nil
}

func (d *Deleter) hasAppAssociation(stackName string) error {
	appMetas, err := d.CF.GetAllApps()
	if err != nil {
		return err
	}

	for _, appMeta := range appMetas {
		for _, app := range appMeta.Apps {
			if app.Lifecycle.Data.Stack == stackName {
				return fmt.Errorf(DeleteStackAppErr, stackName)
			}
		}
	}

	return nil
}

func checkCurlDelete(out, stackName string) error {
	out = strings.Trim(out, " \n")
	var curlErr struct {
		Description string
		ErrorCode   string
		Code        int
	}

	isJSON := strings.HasPrefix(out, "{") && strings.HasSuffix(out, "}")
	if !isJSON {
		return nil
	}

	if err := json.Unmarshal([]byte(out), &curlErr); err != nil {
		return err
	}

	return fmt.Errorf("Failed to delete stack %s with error: %s", stackName, curlErr.Description)
}
