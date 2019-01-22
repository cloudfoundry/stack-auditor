package auditor

import (
	"sort"
	"strings"

	"github.com/cloudfoundry/stack-auditor/cf"
)

type Auditor struct {
	CF cf.CF
}

func (a *Auditor) Audit() (string, error) {
	list, err := a.CF.GetAppsAndStacks()
	if err != nil {
		return "", err
	}

	sort.Strings(list)

	return strings.Join(list, "\n") + "\n", nil
}
