package cf_test

import (
	"fmt"

	"github.com/cloudfoundry/stack-auditor/cf"

	"github.com/cloudfoundry/stack-auditor/mocks"
	"github.com/cloudfoundry/stack-auditor/resources"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF", func() {
	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		c              cf.CF
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockConnection = mocks.NewMockCliConnection(mockCtrl)
		c = cf.CF{Conn: mockConnection}
	})

	When("getAllSpaces", func() {
		It("performs a successful getAllSpaces with empty Json", func() {
			//mockOutput, err := mocks.FileToString("apps.json")
			mockOutput := make([]string, 3)
			var allApps []resources.V3AppsJSON
			cf.V3ResultsPerPage = "1"
			c.Space.Guid = "1234"
			mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?per_page=1")).Return(mockOutput, nil).AnyTimes()
			output, err := c.GetAllApps()
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal(allApps))
		})
	})

	When("CFCurl", func() {
		It("performs a successful CF curl", func() {
			mockOutput, err := mocks.FileToString("apps.json")
			Expect(err).ToNot(HaveOccurred())

			mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps")).Return(mockOutput, nil).AnyTimes()

			output, err := c.CFCurl("/v3/apps")
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal(mockOutput))
		})

		When("given a fully qualified path", func() {
			It("makes it a relative URL", func() {
				mockOutput, err := mocks.FileToString("apps.json")
				Expect(err).NotTo(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/some-endpoint")).Return(mockOutput, nil).AnyTimes()

				output, err := c.CFCurl("https://api.example.com/v3/some-endpoint")
				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(Equal(mockOutput))
			})
		})

		When("hitting a V3 endpoint and CAPI returns an error JSON", func() {
			It("returns the error details in an error", func() {
				mockOutput, err := mocks.FileToString("errorV3.json")
				Expect(err).NotTo(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/some-endpoint")).Return(mockOutput, nil).AnyTimes()

				_, err = c.CFCurl("/v3/some-endpoint")
				Expect(err).To(MatchError("Some V3 error detail, Another V3 error detail"))
			})
		})

		When("hitting a V2 endpoint and CAPI returns an error JSON", func() {
			It("returns the error details in an error", func() {
				mockOutput, err := mocks.FileToString("errorV2.json")
				Expect(err).NotTo(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/some-endpoint")).Return(mockOutput, nil).AnyTimes()

				_, err = c.CFCurl("/v2/some-endpoint")
				Expect(err).To(MatchError("Some error description"))

			})
		})

		When("GetStackGUID", func() {
			It("returns an error when given an invalid stack", func() {
				invalidStack := "NotAStack"
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"stack",
					"--guid",
					invalidStack,
				).Return([]string{}, nil)

				_, err := c.GetStackGUID(invalidStack)
				Expect(err).To(MatchError(invalidStack + " is not a valid stack"))
			})
		})
	})
})
