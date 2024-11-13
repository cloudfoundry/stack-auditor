package changer_test

import (
	"errors"
	"fmt"
	"io"

	plugin_models "code.cloudfoundry.org/cli/plugin/models"

	"github.com/cloudfoundry/stack-auditor/cf"

	"github.com/cloudfoundry/stack-auditor/changer"
	"github.com/cloudfoundry/stack-auditor/mocks"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	AppAName   = "appA"
	AppAGuid   = "appAGuid"
	AppBName   = "appB"
	AppBGuid   = "appBGuid"
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

var _ = Describe("Changer", func() {
	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
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

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("running change-stack", func() {
		It("starts the app after changing stacks", func() {
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

		When("there is an error changing stack metadata", func() {
			It("returns a useful error message", func() {
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

		When("there is an error changing staging on the new stack", func() {
			It("returns a useful error message", func() {
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

				mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
					"curl",
					"/v3/apps/"+AppAGuid+"/actions/start",
					"-X",
					"POST",
				).Return([]string{}, nil)

				_, err := c.ChangeStack(AppAName, StackBName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(changer.ErrorRestagingApp, StackBName))
			})

			When("the app is stopped", func() {
				It("restores the app state", func() {
					mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
						"curl",
						"/v3/apps/"+AppBGuid,
						"-X",
						"PATCH",
						`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackAName+`"} } }`,
					).Return([]string{}, nil)

					restageError := errors.New("restage failed")
					mockRunner.EXPECT().Run("cf", ".", true, "restage", "--strategy", "rolling", AppBName).Return(restageError)

					mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
						"curl",
						"/v3/apps/"+AppBGuid,
						"-X",
						"PATCH",
						`-d={"lifecycle":{"type":"buildpack", "data": {"stack":"`+StackBName+`"} } }`,
					).Return([]string{}, nil)

					mockConnection.EXPECT().CliCommandWithoutTerminalOutput(
						"curl",
						"/v3/apps/"+AppBGuid+"/actions/stop",
						"-X",
						"POST",
					).Return([]string{}, nil)

					_, err := c.ChangeStack(AppBName, StackAName)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(changer.ErrorRestagingApp, StackAName))
				})
			})
		})

		It("returns an error when given the stack that the app is on", func() {
			_, err := c.ChangeStack(AppAName, StackAName)
			Expect(err).To(MatchError("application is already associated with stack " + StackAName))
		})
	})
})
