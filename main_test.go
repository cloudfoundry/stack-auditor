package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
)

const (
	AppAName = "orgA/spaceA/appA"
	AppBName = "orgB/spaceB/appB"
	AppAGuid = "appAGuid"
	AppBGuid = "appBGuid"
	StackAName = "stackA"
	StackBName = "stackB"
	StackAGuid = "stackAGuid"
	StackBGuid = "stackBGuid"
)

func TestRun(t *testing.T) {
	RegisterTestingT(t)

	t.Run("Verify that audit returns the correct stack associations", func(t *testing.T) {
		mockConnection, mockCtrl := setup(t)
		defer mockCtrl.Finish()
		result, err := Audit(mockConnection)
		Expect(err).NotTo(HaveOccurred())

		expectedResult := AppAName + " " + StackAName + "\n" +
			AppBName + " " + StackBName + "\n"
		Expect(result).To(Equal(expectedResult))
	})

	t.Run("Verify that changing stack association uses the right curl", func(t *testing.T){
		mockConnection, mockCtrl := setup(t)
		mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+AppAGuid, "-X", "PUT", `-d={"stack_guid":"`+StackBGuid+`","state":"STOPPED"}`).Return(
			[]string{}, nil)
		mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+AppBGuid, "-X", "PUT", `-d={"stack_guid":"`+StackAGuid+`","state":"STOPPED"}`).Return(
			[]string{}, nil)
		mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+AppAGuid, "-X", "PUT", `-d={"state":"STARTED"}`).Return(
			[]string{}, nil)
		mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps/"+AppBGuid, "-X", "PUT", `-d={"state":"STARTED"}`).Return(
			[]string{}, nil)

		defer mockCtrl.Finish()

		result, err := ChangeStack(mockConnection, AppAName, StackBName)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(fmt.Sprintf(ChangeStackSuccessMsg, AppAName, StackBName)))

		result, err = ChangeStack(mockConnection, AppBName, StackAName)
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(fmt.Sprintf(ChangeStackSuccessMsg, AppBName, StackAName)))
	})

	t.Run("Verify that giving an incorrect stack name returns an error", func(t *testing.T) {
		mockConnection, mockCtrl := setup(t)

		defer mockCtrl.Finish()
		_, err := ChangeStack(mockConnection, AppAName, "WrongStack")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("WrongStack is not a valid stack"))
	})

	t.Run("Verify that giving a stack which you are already on returns an error", func(t *testing.T) {
		mockConnection, mockCtrl := setup(t)

		defer mockCtrl.Finish()
		_, err := ChangeStack(mockConnection, AppAName, StackAName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("application is already associated with stack " + StackAName))
	})

	t.Run("Verify that giving app name which doesn't have org and space will return an error", func(t *testing.T) {
		mockConnection, mockCtrl := setup(t)

		defer mockCtrl.Finish()
		_, err := ChangeStack(mockConnection, "appA", StackAName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("invalid App Name – doesn't have Org/Space/App"))
	})

	t.Run("Verify that giving app which can't be found returns an error", func(t *testing.T) {
		mockConnection, mockCtrl := setup(t)

		defer mockCtrl.Finish()
		fakeApp := "orgC/spaceC/appC"
		_, err := ChangeStack(mockConnection, fakeApp, StackAName)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("application could not be found"))
	})
}

func setup(t *testing.T) (*MockCliConnection, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)

	apps, err := fileToString("apps.json")
	Expect(err).ToNot(HaveOccurred())
	spaces, err := fileToString("spaces.json")
	Expect(err).ToNot(HaveOccurred())
	stacks, err := fileToString("stacks.json")
	Expect(err).ToNot(HaveOccurred())

	mockConnection := NewMockCliConnection(mockCtrl)
	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps").Return(
		[]string{
			apps,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/spaces").Return(
		[]string{
			spaces,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/stacks").Return(
		[]string{
			stacks,
		}, nil).AnyTimes()

	mockConnection.EXPECT().GetOrgs().Return(
		[]plugin_models.GetOrgs_Model{
			plugin_models.GetOrgs_Model{
				Guid: "orgAGuid",
				Name: "orgA",
			},

			plugin_models.GetOrgs_Model{
				Guid: "orgBGuid",
				Name: "orgB",
			},
		}, nil).AnyTimes()

	return mockConnection, mockCtrl
}

func fileToString(fileName string) (string, error) {
	path, err := filepath.Abs(filepath.Join("fixtures", fileName))
	if err != nil {
		return "", err
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}
