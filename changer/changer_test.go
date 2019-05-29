package changer_test

import (
	"fmt"
	"io"
	"testing"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"

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
	AppAGuid   = "appAGuid"
	StackAName = "stackA"
	StackBName = "stackB"
)

//go:generate mockgen -source=changer.go -destination=mocks_test.go -package=changer_test

var (
	mockCtrl       *gomock.Controller
	mockConnection *mocks.MockCliConnection
	mockRunner     *MockRunner
	c              changer.Changer
	logMsg         string
)

func TestUnitChanger(t *testing.T) {
	spec.Run(t, "Changer", testChanger, spec.Report(report.Terminal{}))
}

func testChanger(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)

		mockCtrl = gomock.NewController(t)
		mockConnection = mocks.SetupMockCliConnection(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)

		c = changer.Changer{
			Runner: mockRunner,
			CF: cf.CF{
				Conn: mockConnection,
				Space: plugin_models.Space{
					plugin_models.SpaceFields{
						Guid: mocks.SpaceGuid,
						Name: mocks.SpaceName,
					},
				},
			},
			Log: func(w io.Writer, msg string) {
				logMsg = msg
			},
		}

	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("running change-stack", func() {
		when("when you have zero down time endpoint available and the --v3 flag is set", func() {
			it("starts the app after changing stacks", func() {
				c.V3Flag = true

				setupV3Restart()

				mockRunner.EXPECT().Run("cf", ".", true, "v3-zdt-restart", AppAName)

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/actions/start",
					"-X", "POST")

				result, err := c.ChangeStack(AppAName, StackBName)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))
			})
		})

		when("when you have zero down time endpoint available and the --v3 flag is NOT set", func() {
			it("starts the app after changing stacks", func() {
				c.V3Flag = false

				setupV3Restart()

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/appAGuid/actions/restart",
					"-X", "POST",
				).Return([]string{}, nil)

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/actions/start",
					"-X", "POST")

				result, err := c.ChangeStack(AppAName, StackBName)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))
			})
		})

		when("when you do not have zero down time endpoint available", func() {
			it("starts the app after changing stacks when the --v3 flag is NOT set", func() {
				c.V3Flag = false

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

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/actions/start",
					"-X", "POST")

				result, err := c.ChangeStack(AppAName, StackBName)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))
			})

			it("errors out when the --v3 flag is set", func() {
				c.V3Flag = true
				mockConnection.EXPECT().ApiVersion().Return("0.0.1", nil).AnyTimes()
				_, err := c.ChangeStack(AppAName, StackBName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(changer.ErrorZDTNotSupported))
			})
		})

		when("there is an error changing stack metadata", func() {
			it("returns a useful error message", func() {
				errorMsg, err := mocks.FileToString("lifecycleV3Error.json")
				Expect(err).ToNot(HaveOccurred())
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid,
					"-X",
					"PATCH",
					`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
				).Return(errorMsg, nil)

				mockConnection.EXPECT().ApiVersion().Return("99.99.99", nil).AnyTimes()

				_, err = c.ChangeStack(AppAName, StackBName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(changer.ErrorChangingStack, StackBName))
			})
		})

		it("returns an error when given the stack that the app is on", func() {
			_, err := c.ChangeStack(AppAName, StackAName)
			Expect(err).To(MatchError("application is already associated with stack " + StackAName))
		})
	})
}

func setupV3Restart() {
	mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
		"curl",
		"/v3/apps/"+AppAGuid,
		"-X",
		"PATCH",
		`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
	).Return([]string{}, nil)

	mockConnection.EXPECT().ApiVersion().Return("99.99.99", nil)

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
}
