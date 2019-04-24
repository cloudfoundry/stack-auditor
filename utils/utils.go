package utils

import (
	"encoding/json"

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
