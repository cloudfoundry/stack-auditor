package cf_test

import (
	"fmt"
	"testing"

	"github.com/cloudfoundry/stack-auditor/cf"

	"github.com/cloudfoundry/stack-auditor/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitCF(t *testing.T) {
	spec.Run(t, "CF", testCF, spec.Report(report.Terminal{}))
}

func testCF(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		c              cf.CF
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockConnection = mocks.NewMockCliConnection(mockCtrl)
		c = cf.CF{Conn: mockConnection}
	})

	when("CFCurl", func() {
		it("performs a successful CF curl", func() {
			mockOutput, err := mocks.FileToString("apps.json")
			Expect(err).ToNot(HaveOccurred())

			mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps")).Return(mockOutput, nil).AnyTimes()

			output, err := c.CFCurl("/v3/apps")
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(Equal(mockOutput))
		})

		when("given a fully qualified path", func() {
			it("makes it a relative URL", func() {
				mockOutput, err := mocks.FileToString("apps.json")
				Expect(err).NotTo(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/some-endpoint")).Return(mockOutput, nil).AnyTimes()

				output, err := c.CFCurl("https://api.example.com/v3/some-endpoint")
				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(Equal(mockOutput))
			})
		})

		when("hitting a V3 endpoint and CAPI returns an error JSON", func() {
			it("returns the error details in an error", func() {
				mockOutput, err := mocks.FileToString("errorV3.json")
				Expect(err).NotTo(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/some-endpoint")).Return(mockOutput, nil).AnyTimes()

				_, err = c.CFCurl("/v3/some-endpoint")
				Expect(err).To(MatchError("Some V3 error detail, Another V3 error detail"))
			})
		})

		when("hitting a V2 endpoint and CAPI returns an error JSON", func() {
			it("returns the error details in an error", func() {
				mockOutput, err := mocks.FileToString("errorV2.json")
				Expect(err).NotTo(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/some-endpoint")).Return(mockOutput, nil).AnyTimes()

				_, err = c.CFCurl("/v2/some-endpoint")
				Expect(err).To(MatchError("Some error description"))

			})
		})

		when("GetStackGUID", func() {
			it("returns an error when given an invalid stack", func() {
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
}
