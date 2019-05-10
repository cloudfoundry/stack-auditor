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
		when("the app is initially stopped", func() {
			it("does not start the app after changing stacks", func() {
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+AppBGuid, "-X", "PUT", `-d={"stack_guid":"`+StackAGuid+`","state":"STOPPED"}`).Return(
					[]string{}, nil)

				result, err := c.ChangeStack(AppBName, StackAName, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppBName, StackAName)))
			})
		})

		//		when("with zero downtime", func() {
		//			it("makes a call to the v3 endpoint", func() {
		//				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v3/apps/"+AppBGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackAName+`"}}}`).Return(
		//					[]string{}, nil)
		//				result, err := c.ChangeStack(AppBName, StackAName, true)
		//				Expect(err).NotTo(HaveOccurred())
		//				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppBName, StackAName)))
		//			})
		//
		//			it("returns an error when v3 is not supported", func() {
		//				errorString := `{
		//   "errors": [
		//      {
		//         "detail": "Unknown request",
		//         "title": "CF-NotFound",
		//         "code": 10000
		//      }
		//   ]
		//}`
		//
		//				mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v3/apps/"+AppBGuid, "-X", "PATCH", `-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackAName+`"}}}`).Return(
		//					[]string{errorString}, nil)
		//				_, err := c.ChangeStack(AppBName, StackAName, true)
		//				Expect(err).To(HaveOccurred())
		//				Expect(err.Error()).To(ContainSubstring(changer.ChangeStackV3ErrorMsg))
		//			})
		//		})

		when.Focus("when you have zero down time endpoint available", func() {
			it("starts the app after changing stacks", func() {

				// Load return strings
				v3AppJSON, err := mocks.FileToString("appV3.json")
				Expect(err).ToNot(HaveOccurred())

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid,
					"-X",
					"PATCH",
					`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"}}}`,
				).Return(v3AppJSON, nil)

				v3PackageJSON := []string{""}
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/packages",
					"-X",
					"GET",
				).Return(v3PackageJSON, nil)

				mockConnection.EXPECT().CliCommand("v3-stage test-app", "--package-guid", "123")

				v3DropletJSON, err := mocks.FileToString("appV3Droplets.json")

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/droplets",
					"-X",
					"GET",
				).Return(v3DropletJSON, nil)

				mockRunner.EXPECT().Run("cf", ".", false, "v3-set-droplet", AppAName, "--droplet-guid", "123")

				mockRunner.EXPECT().Run("cf", ".", false, "v3-zdt-restart", AppAName)

				result, err := c.ChangeStack(AppAName, StackBName, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))

			})

		})

		when("when you do not have zero down time endpoint available", func() {

		})

		when("changing the stack of an app on the v3 endpoint that doesn't exist", func() {
			it("returns the error from the output of the curl command", func() {
				_, err := c.ChangeStack(NotAnApp, StackBName, false)
				Expect(err).To(MatchError("Some V3 error detail, Another V3 error detail"))
			})
		})

		it("returns an error when given an invalid stack", func() {
			_, err := c.ChangeStack(AppAName, "WrongStack", false)
			Expect(err).To(MatchError("WrongStack is not a valid stack"))
		})

		it("returns an error when given the stack that the app is on", func() {
			_, err := c.ChangeStack(AppAName, StackAName, false)
			Expect(err).To(MatchError("application is already associated with stack " + StackAName))
		})

		it("returns an error when an app can't be found", func() {
			fakeApp := "appC"
			_, err := c.ChangeStack(fakeApp, StackAName, false)
			Expect(err).To(MatchError("no app found with name " + fakeApp))
		})
	})
}
