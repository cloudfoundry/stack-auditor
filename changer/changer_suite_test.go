package changer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChanger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Changer Suite")
}
