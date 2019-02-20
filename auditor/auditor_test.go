package auditor_test

import (
	"testing"

	"github.com/cloudfoundry/stack-auditor/auditor"
	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/cloudfoundry/stack-auditor/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const (
	AppAName   = "appA"
	AppBName   = "appB"
	AppAPath   = "commonOrg/commonSpace/" + AppAName
	AppBPath   = "commonOrg/commonSpace/" + AppBName
	StackAName = "stackA"
	StackBName = "stackB"
)

func TestUnitAuditor(t *testing.T) {
	spec.Run(t, "Audit", testAudit, spec.Report(report.Terminal{}))
}

func testAudit(t *testing.T, when spec.G, it spec.S) {

	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		a              auditor.Auditor
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)

		mockConnection = mocks.SetupMockCliConnection(mockCtrl)

		a = auditor.Auditor{
			CF: cf.CF{
				Conn: mockConnection,
			},
		}
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("running audit-stack", func() {
		it("Verify that cf returns the correct stack associations", func() {
			result, err := a.Audit()
			Expect(err).NotTo(HaveOccurred())

			expectedResult := AppAPath + " " + StackAName + "\n" +
				AppBPath + " " + StackBName + "\n"
			Expect(result).To(Equal(expectedResult))
		})
	})
}
