package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

const (
	oldStack = "cflinuxfs2"
	newStack = "cflinuxfs3"
)

func TestIntegrationStackAuditor(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	when("Change Stack", func() {
		var (
			app *cutlass.App
		)

		it.Before(func() {
			app = cutlass.New(filepath.Join("testdata", "simple_app"))
			app.Stack = oldStack
		})

		it.After(func() {
			if app != nil {
				Expect(app.Destroy()).To(Succeed())
			}
			app = nil
		})

		it("should change the stack", func() {
			PushAppAndConfirm(app)
			cmd := exec.Command("cf", "change-stack", app.Name, newStack)
			output, err := cmd.Output()
			Expect(output).To(ContainSubstring("Starting app %s", app.Name))
			Expect(err).ToNot(HaveOccurred())

			cmd = exec.Command("cf", "app", app.Name)
			contents, err := cmd.Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(contents).To(ContainSubstring(newStack))
		})
	})

	when("Audit Stack", func() {
		const appCount = 51 //50 apps per page
		var (
			apps               [appCount]*cutlass.App
			spaceName, orgName string
			err                error
			stacks             = []string{"cflinuxfs2", "cflinuxfs3"}
		)

		it.Before(func() {
			orgName, spaceName, err = GetOrgAndSpace()
			Expect(err).ToNot(HaveOccurred())

			wg := sync.WaitGroup{}
			wg.Add(appCount)
			for i := 0; i < appCount; i++ {
				apps[i] = cutlass.New(filepath.Join("testdata", "simple_app"))
				apps[i].Stack = stacks[i%len(stacks)]

				go func(i int) {
					defer wg.Done()
					PushAppAndConfirm(apps[i])
				}(i)
			}
			wg.Wait()
		})

		it.After(func() {
			for _, app := range apps {
				app.Destroy()
			}
		})

		it("prints all apps with their orgs spaces and stacks", func() {
			cmd := exec.Command("cf", "audit-stack")
			output, err := cmd.Output()
			Expect(err).NotTo(HaveOccurred())

			for i, app := range apps {
				Expect(string(output)).To(ContainSubstring(fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, app.Name, stacks[i%len(stacks)])))
			}
		})
	})
}

func PushAppAndConfirm(app *cutlass.App) {
	Expect(app.Push()).To(Succeed())
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
}

func GetOrgAndSpace() (string, string, error) {
	cfHome := os.Getenv("CF_HOME")
	if cfHome == "" {
		cfHome = os.Getenv("HOME")
	}
	bytes, err := ioutil.ReadFile(filepath.Join(cfHome, ".cf", "config.json"))
	if err != nil {
		return "", "", err
	}

	var configData struct {
		SpaceFields struct {
			Name string
		}
		OrganizationFields struct {
			Name string
		}
	}

	if err := json.Unmarshal(bytes, &configData); err != nil {
		return "", "", err
	}
	return configData.OrganizationFields.Name, configData.SpaceFields.Name, nil
}
