package utils

import (
	"encoding/json"
	"strings"

	"code.cloudfoundry.org/cli/cf/errors"

	"github.com/cloudfoundry/stack-auditor/resources"
)

func CheckOutputForErrorMessage(output, errorMsg string) bool {
	var errorsJSON resources.ErrorsJson
	if err := json.Unmarshal([]byte(output), &errorsJSON); err != nil {
		return false
	}

	if len(errorsJSON.Errors) > 0 && errorsJSON.Errors[0].Title == errorMsg {
		return true
	}
	return false
}

func CheckV3Error(lines []string) error {
	output := strings.Join(lines, "\n")
	var errorsJSON resources.ErrorsJson

	if err := json.Unmarshal([]byte(output), &errorsJSON); err != nil {
		return nil
	}

	errorDetails := make([]string, 0)
	for _, e := range errorsJSON.Errors {
		errorDetails = append(errorDetails, e.Detail)
	}

	return errors.New(strings.Join(errorDetails, ", "))
}
