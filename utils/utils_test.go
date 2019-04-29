package utils_test

import (
	"strings"
	"testing"

	"github.com/cloudfoundry/stack-auditor/mocks"
	"github.com/cloudfoundry/stack-auditor/utils"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitUtils(t *testing.T) {
	spec.Run(t, "Utils", testUtils, spec.Report(report.Terminal{}))
}

func testUtils(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("CheckV3Error", func() {
		when("the lines of text form a V3 JSON error", func() {
			it("returns the error details in an error", func() {
				input, err := mocks.FileToString("errorV3.json")
				Expect(err).NotTo(HaveOccurred())
				err = utils.CheckV3Error(strings.Split(input, "\n"))
				Expect(err).To(MatchError("Some V3 error detail, Another V3 error detail"))
			})
		})

		when("the lines of text do not form a V3 JSON error", func() {
			it("does not return an error", func() {
				input := []string{"not an error"}
				Expect(utils.CheckV3Error(input)).To(Succeed())
			})
		})
	})
}
