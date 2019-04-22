package mocks

import (
	"fmt"
	"github.com/cloudfoundry/stack-auditor/cf"
	"io/ioutil"
	"path/filepath"

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
)

func SetupMockCliConnection(mockCtrl *gomock.Controller) *MockCliConnection {
	apps, err := fileToString("apps.json")
	Expect(err).ToNot(HaveOccurred())
	spaces, err := fileToString("spaces.json")
	Expect(err).ToNot(HaveOccurred())
	buildpacks, err := fileToString("buildpacks.json")
	Expect(err).ToNot(HaveOccurred())

	mockConnection := NewMockCliConnection(mockCtrl)
	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v3/apps?per_page=%d", cf.V3ResultsPerPage)).Return(
		[]string{
			apps,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", fmt.Sprintf("/v2/spaces?results-per-page=%d", cf.V2ResultsPerPage)).Return(
		[]string{
			spaces,
		}, nil).AnyTimes()

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
		[]string{
			buildpacks,
		}, nil).AnyTimes()

	mockConnection.EXPECT().GetOrgs().Return(
		[]plugin_models.GetOrgs_Model{
			plugin_models.GetOrgs_Model{
				Guid: "commonOrgGuid",
				Name: "commonOrg",
			},

			plugin_models.GetOrgs_Model{
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

func fileToString(fileName string) (string, error) {
	path, err := filepath.Abs(filepath.Join("..", "integration", "testdata", fileName))
	if err != nil {
		return "", err
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}
