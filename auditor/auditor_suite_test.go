package auditor_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuditor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auditor Suite")
}
