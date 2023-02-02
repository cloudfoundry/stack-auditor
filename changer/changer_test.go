package changer_test

import (
	"fmt"
	"io"
	"testing"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"

	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/pkg/errors"

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
		it("starts the app after changing stacks", func() {
			mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
				"curl",
				"/v3/apps/"+AppAGuid,
				"-X",
				"PATCH",
				`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
			).Return([]string{}, nil)

			mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
				"curl",
				"/v3/apps/"+AppAGuid+"/actions/start",
				"-X",
				"POST",
			)

			mockRunner.EXPECT().Run("cf", ".", true, "restage", "--strategy", "rolling", AppAName)

			result, err := c.ChangeStack(AppAName, StackBName)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(fmt.Sprintf(changer.ChangeStackSuccessMsg, AppAName, StackBName)))
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

				_, err = c.ChangeStack(AppAName, StackBName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(changer.ErrorChangingStack, StackBName))
			})
		})

		when("there is an error changing staging on the new stack", func() {
			it("returns a useful error message", func() {
				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid,
					"-X",
					"PATCH",
					`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
				).Return([]string{}, nil)

				restageError := errors.New("restage failed")
				mockRunner.EXPECT().Run("cf", ".", true, "restage", "--strategy", "rolling", AppAName).Return(restageError)

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid,
					"-X",
					"PATCH",
					`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackAName+`"} } }`,
				).Return([]string{}, nil)

				_, err := c.ChangeStack(AppAName, StackBName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(changer.ErrorRestagingApp, StackBName))
			})
		})

		it("returns an error when given the stack that the app is on", func() {
			_, err := c.ChangeStack(AppAName, StackAName)
			Expect(err).To(MatchError("application is already associated with stack " + StackAName))
		})
	})
}
