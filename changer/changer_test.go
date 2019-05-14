package changer_test

import (
	"fmt"
	"testing"

	"github.com/cloudfoundry/stack-auditor/cf"

	"github.com/cloudfoundry/stack-auditor/changer"
	"github.com/cloudfoundry/stack-auditor/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

const (
	AppAName   = "appA"
	AppBName   = "appB"
	AppAGuid   = "appAGuid"
	AppBGuid   = "appBGuid"
	StackAName = "stackA"
	StackBName = "stackB"
	StackAGuid = "stackAGuid"
	StackBGuid = "stackBGuid"
	NotAnApp   = "notAnApp"
)

//go:generate mockgen -source=changer.go -destination=mocks_test.go -package=changer_test

func TestUnitChanger(t *testing.T) {
	spec.Run(t, "Changer", testChanger, spec.Report(report.Terminal{}))
}

func testChanger(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		mockRunner     *MockRunner
		c              changer.Changer
	)

	it.Before(func() {
		RegisterTestingT(t)

		mockCtrl = gomock.NewController(t)
		mockConnection = mocks.SetupMockCliConnection(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)

		c = changer.Changer{
			Runner: mockRunner,
			CF: cf.CF{
				Conn: mockConnection,
			},
		}
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("running change-stack", func() {
		when("when you have zero down time endpoint available", func() {
			it("starts the app after changing stacks", func() {
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid,
					"-X",
					"PATCH",
					`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
				).Return([]string{}, nil)

				mockConnection.EXPECT().ApiVersion().Return("99.99.99", nil)

				mockRunner.EXPECT().Run("cf", ".", false, "v3-zdt-restart", AppAName)

				result, err := c.ChangeStack(AppAName, StackBName, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))
			})
		})

		when("when you do not have zero down time endpoint available", func() {
			it("starts the app after changing stacks", func() {
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid,
					"-X",
					"PATCH",
					`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
				).Return([]string{}, nil)

				mockConnection.EXPECT().ApiVersion().Return("0.0.1", nil)

				appADroplet, err := mocks.FileToString("appV3Droplet.json")
				Expect(err).ToNot(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/droplets/current",
				).Return(appADroplet, nil)

				appABuildPost, err := mocks.FileToString("appAV3BuildPost.json")
				Expect(err).ToNot(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/builds",
					"-X", "POST",
					`-d='{"package": {"guid": "og-package-guid"} }'`,
				).Return(appABuildPost, nil)

				appABuildGet, err := mocks.FileToString("appAV3BuildGet.json")
				Expect(err).ToNot(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/builds/some-build-guid",
				).Return(appABuildGet, nil)

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/appAGuid/relationships/current_droplet",
					"-X", "PATCH",
					`-d='{ "data": { "guid": "some-droplet-guid" } }'`,
				).Return([]string{}, nil)


				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/appAGuid/actions/restart",
					"-X", "POST",
				).Return([]string{}, nil)

				result, err := c.ChangeStack(AppAName, StackBName, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))
			})
		})

		it("returns an error when given the stack that the app is on", func() {
			_, err := c.ChangeStack(AppAName, StackAName, false)
			Expect(err).To(MatchError("application is already associated with stack " + StackAName))
		})
	})
}
