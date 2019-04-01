package mocks

import (
	"io/ioutil"
	"path/filepath"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -package mocks -destination cli_connection.go code.cloudfoundry.org/cli/plugin CliConnection

func SetupMockCliConnection(mockCtrl *gomock.Controller) *MockCliConnection {
	apps, err := fileToString("apps.json")
	Expect(err).ToNot(HaveOccurred())
	spaces, err := fileToString("spaces.json")
	Expect(err).ToNot(HaveOccurred())
	stacks, err := fileToString("stacks.json")
	Expect(err).ToNot(HaveOccurred())
	buildpacks, err := fileToString("buildpacks.json")
	Expect(err).ToNot(HaveOccurred())

	mockConnection := NewMockCliConnection(mockCtrl)
	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/apps?results-per-page=100").Return(
		[]string{
			apps,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/spaces?results-per-page=100").Return(
		[]string{
			spaces,
		}, nil).AnyTimes()

	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/stacks?results-per-page=100").Return(
		[]string{
			stacks,
		}, nil).AnyTimes()
	mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/buildpacks?results-per-page=100").Return(
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

	SetCurrentOrgAndSpace(mockConnection, "commonOrg", "commonSpace")

	return mockConnection
}

func SetCurrentOrgAndSpace(mockConnection *MockCliConnection, org string, space string) {
	mockConnection.EXPECT().GetCurrentOrg().Return(plugin_models.Organization{
		OrganizationFields: plugin_models.OrganizationFields{
			Name: org},
	}, nil).AnyTimes()
	mockConnection.EXPECT().GetCurrentSpace().Return(plugin_models.Space{
		SpaceFields: plugin_models.SpaceFields{
			Name: space},
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
