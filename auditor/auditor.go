package auditor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cloudfoundry/stack-auditor/cf"
)

const (
	AuditStackMsg = "Retrieving stack information for all apps...\n\n"
)

type Auditor struct {
	CF cf.CF
}

func (a *Auditor) Audit() (string, error) {
	fmt.Printf(AuditStackMsg)

	list, err := a.CF.GetAppsAndStacks()
	if err != nil {
		return "", err
	}

	sort.Strings(list)

	return strings.Join(list, "\n") + "\n", nil
}
