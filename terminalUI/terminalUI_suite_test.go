package terminalUI_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTerminalUI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TerminalUI Suite")
}
