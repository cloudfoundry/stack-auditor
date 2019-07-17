package auditor

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AuditStackMsg = "Retrieving stack information for all apps...\n\n"
	JSONFlag      = "json"
	CSVFlag       = "csv"
)

type Auditor struct {
	CF         cf.CF
	OutputType string
}

func (a *Auditor) Audit() (string, error) {
	if a.OutputType == "" {
		fmt.Printf(AuditStackMsg)
	}

	apps, err := a.CF.GetAppsAndStacks()
	if err != nil {
		return "", err
	}

	sort.Sort(apps)

	if a.OutputType == CSVFlag {
		return apps.CSV()
	}
	if a.OutputType == JSONFlag {
		json, err := json.Marshal(apps)
		if err != nil {
			return "", nil
		}
		return string(json), nil
	}

	return fmt.Sprintf("%s", apps), nil
}
