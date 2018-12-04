package main

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"code.cloudfoundry.org/cli/plugin/models"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
)

func TestRun(t *testing.T) {
	RegisterTestingT(t)
	t.Run("Runs with args", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

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
			}, nil)

		mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/spaces").Return(
			[]string{
				spaces,
			}, nil)

		mockConnection.EXPECT().CliCommandWithoutTerminalOutput("curl", "/v2/stacks").Return(
			[]string{
				stacks,
			}, nil)

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
			}, nil)

		result, err := Audit(mockConnection)
		Expect(err).NotTo(HaveOccurred())

		expectedResult := "orgA/spaceA/appA stackA\norgB/spaceB/appB stackB\n"
		Expect(result).To(Equal(expectedResult))
	})
}

func fileToString(fileName string) (string, error) {
	path, err := filepath.Abs(filepath.Join("resources", fileName))
	if err != nil {
		return "", err
	}

	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}
