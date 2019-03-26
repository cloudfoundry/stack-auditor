package deleter_test

import (
	"fmt"
	"testing"

	"github.com/cloudfoundry/stack-auditor/cf"

	"github.com/cloudfoundry/stack-auditor/deleter"
	"github.com/cloudfoundry/stack-auditor/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const (
	StackAName   = "stackA"
	StackAGuid   = "stackAGuid"
	InvalidStack = "notarealstack"
)

func TestUnitDeleter(t *testing.T) {
	spec.Run(t, "Deleter", testDeleter, spec.Report(report.Terminal{}))
}

func testDeleter(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		d              deleter.Deleter
	)

	it.Before(func() {
		RegisterTestingT(t)

		mockCtrl = gomock.NewController(t)
		mockConnection = mocks.SetupMockCliConnection(mockCtrl)

		d = deleter.Deleter{
			CF: cf.CF{
				Conn: mockConnection,
			},
		}
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("deleting a stack that no apps are using", func() {
		it("deletes the stack", func() {

			mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/stacks/"+StackAGuid, "-X", "DELETE").Return([]string{}, nil)
			result, err := d.DeleteStack(StackAName)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(fmt.Sprintf("Stack %s has been deleted", StackAName)))
		})
	})

	when("deleting a stack that does not exist", func() {
		it("should tell the user the stack is invalid", func() {
			_, err := d.DeleteStack(InvalidStack)
			Expect(err).To(MatchError(fmt.Sprintf("%s is not a valid stack", InvalidStack)))
		})
	})
}
