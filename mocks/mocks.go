package mocks

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/stack-auditor/cf"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -package mocks -destination cli_connection.go code.cloudfoundry.org/cli/plugin CliConnection
var (
	StackAName = "stackA"
	StackBName = "stackB"
	StackAGuid = "stackAGuid"
	StackBGuid = "stackBGuid"
	StackEName = "stackE"
	StackEGuid = "stackEGuid"
	AppAName   = "appA"
	AppBName   = "appB"
	AppCName   = "appC"
	NotAnApp   = "notAnApp"
)

func SetupMockCliConnection(mockCtrl *gomock.Controller) *MockCliConnection {
	apps, err := FileToString("apps.json")
	Expect(err).ToNot(HaveOccurred())

	appA, err := FileToString("appA.json")
	Expect(err).ToNot(HaveOccurred())

	appB, err := FileToString("appB.json")
	Expect(err).ToNot(HaveOccurred())

	appC, err := FileToString("appC.json")
	Expect(err).ToNot(HaveOccurred())

	spaces, err := FileToString("spaces.json")
	Expect(err).ToNot(HaveOccurred())

	buildpacks, err := FileToString("buildpacks.json")
	Expect(err).ToNot(HaveOccurred())

	errorV3, err := FileToString("errorV3.json")
	Expect(err).ToNot(HaveOccurred())

	mockConnection := NewMockCliConnection(mockCtrl)
	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?per_page=%d", cf.V3ResultsPerPage)).Return(
		apps, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?names=%s", AppAName)).Return(
		appA,
		nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?names=%s", AppBName)).Return(
		appB,
		nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?names=%s", AppCName)).Return(
		appC,
		nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?names=%s", NotAnApp)).Return(errorV3, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/spaces?results-per-page=%d", cf.V2ResultsPerPage)).Return(
		spaces,
		nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("stack", "--guid", StackAName).Return(
		[]string{
			StackAGuid,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("stack", "--guid", StackBName).Return(
		[]string{
			StackBGuid,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("stack", "--guid", StackEName).Return(
		[]string{
			StackEGuid,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("stack", "--guid", gomock.Any()).Return(
		[]string{}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/buildpacks?results-per-page=%d", cf.V2ResultsPerPage)).Return(
		buildpacks,
		nil).AnyTimes()

	mockConnection.EXPECT().GetOrgs().Return(
		[]plugin_models.GetOrgs_Model{
			{
				Guid: "commonOrgGuid",
				Name: "commonOrg",
			},

			{
				Guid: "orgBGuid",
				Name: "orgB",
			},
		}, nil).AnyTimes()

	SetCurrentOrgAndSpace(mockConnection, "commonOrg", "commonSpace", "commonSpaceGuid")

	return mockConnection
}

func SetCurrentOrgAndSpace(mockConnection *MockCliConnection, org string, space string, spaceGuid string) {
	mockConnection.EXPECT().GetCurrentOrg().Return(plugin_models.Organization{
		OrganizationFields: plugin_models.OrganizationFields{
			Name: org},
	}, nil).AnyTimes()
	mockConnection.EXPECT().GetCurrentSpace().Return(plugin_models.Space{
		SpaceFields: plugin_models.SpaceFields{
			Name: space, Guid: spaceGuid},
	}, nil).AnyTimes()
}

// TODO move this somewhere more appropriate
func FileToString(fileName string) ([]string, error) {
	path, err := filepath.Abs(filepath.Join("..", "testdata", fileName))
	if err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(buf), "\n"), nil
}
