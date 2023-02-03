package deleter_test

import (
	"fmt"

	"github.com/cloudfoundry/stack-auditor/cf"

	"github.com/cloudfoundry/stack-auditor/deleter"
	"github.com/cloudfoundry/stack-auditor/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	StackEName   = "stackE"
	StackEGuid   = "stackEGuid"
	StackCName   = "stackC"
	InvalidStack = "notarealstack"
)

var _ = Describe("Deleter", func() {
	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		d              deleter.Deleter
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockConnection = mocks.SetupMockCliConnection(mockCtrl)

		d = deleter.Deleter{
			CF: cf.CF{
				Conn: mockConnection,
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("deleting a stack that no apps are using", func() {
		It("deletes the stack", func() {
			mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/stacks/"+StackEGuid, "-X", "DELETE").Return([]string{}, nil)
			result, err := d.DeleteStack(StackEName)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring(fmt.Sprintf("Stack %s has been deleted", StackEName)))
		})
	})

	When("deleting a stack that does not exist", func() {
		It("should tell the user the stack is invalid", func() {
			_, err := d.DeleteStack(InvalidStack)
			Expect(err).To(MatchError(fmt.Sprintf("%s is not a valid stack", InvalidStack)))
		})
	})

	When("deleting a stack that has buildpacks associated with it", func() {
		It("should tell the user to the delete the buildpack first", func() {
			_, err := d.DeleteStack(StackCName)
			Expect(err).To(MatchError(fmt.Sprintf(deleter.DeleteStackBuildpackErr, StackCName)))
		})
	})
})
