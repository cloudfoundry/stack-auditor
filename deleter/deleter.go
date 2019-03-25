package deleter

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/stack-auditor/cf"
	"strings"
)

const DeleteStackSuccessMsg = "Stack %s has been deleted"

type Deleter struct {
	CF cf.CF
}

func (d *Deleter) DeleteStack(stackName string) (string, error) {
	stackGuid, err := d.CF.GetStackGUID(stackName)
	if err != nil {
		return "", err
	}

	lines, err := d.CF.Conn.CliCommandWithoutTerminalOutput("curl", "/v2/stacks/"+stackGuid, "-X", "DELETE")
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

func checkCurlDelete(out, stackName string) error {

	out = strings.Trim(out," \n")
	var curlErr struct {
		Description string
		ErrorCode  string
		Code   int

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
