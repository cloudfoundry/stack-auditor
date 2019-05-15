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

	"github.com/cloudfoundry/stack-auditor/changer"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

const (
	oldStack            = "cflinuxfs2"
	newStack            = "cflinuxfs3"
	fakeStack           = "fakeStack"
	fakeBuildpack       = "fakeBuildpack"
	oldStackDescription = "Cloud Foundry Linux-based filesystem (Ubuntu 14.04)"
	newStackDescription = "Cloud Foundry Linux-based filesystem (Ubuntu 18.04)"
	appBody             = "Hello World!"
	duration            = 90 * time.Second
	interval            = 100 * time.Millisecond
	disk                = "128M"
	memory              = "128M"
)

func TestIntegrationStackAuditor(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when.Focus("Change Stack", func() {
		var (
			app *cutlass.App
		)

		when("the app was initially started", func() {
			it.Before(func() {
				Expect(CreateStack(oldStack, oldStackDescription)).To(Succeed())
				Expect(CreateStack(newStack, newStackDescription)).To(Succeed())
				app = cutlass.New(filepath.Join("testdata", "simple_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory
			})

			it("should change the stack and remain started", func() {
				PushAppAndConfirm(app, true)
				defer app.Destroy()

				cmd := exec.Command("cf", "change-stack", app.Name, newStack)
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring(changer.RestoringStateMsg, "STARTED"))

				cmd = exec.Command("cf", "app", app.Name)
				contents, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				Expect(contents).To(ContainSubstring(newStack))
			})
		})

		when("the app was initially stopped", func() {
			it.Before(func() {
				Expect(CreateStack(oldStack, oldStackDescription)).To(Succeed())
				Expect(CreateStack(newStack, newStackDescription)).To(Succeed())
				app = cutlass.New(filepath.Join("testdata", "simple_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory
			})

			it("it should change the stack and remain stopped", func() {
				PushAppAndConfirm(app, false)
				defer app.Destroy()

				cmd := exec.Command("cf", "change-stack", app.Name, newStack)
				out, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred(), string(out))
				Expect(string(out)).To(ContainSubstring(changer.RestoringStateMsg, "STOPPED"))

				cmd = exec.Command("cf", "app", app.Name)
				contents, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(newStack))
				Expect(string(contents)).To(MatchRegexp(`requested state:\s*stopped`))
			})
		})

		when("the app cannot stage on the target stack", func() {
			it.Before(func() {
				Expect(CreateStack(oldStack, oldStackDescription)).To(Succeed())
				Expect(CreateStack(newStack, newStackDescription)).To(Succeed())
				app = cutlass.New(filepath.Join("testdata", "fs2_only_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/ruby-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory
			})

			it("restarts itself on the old stack", func() {
				PushAppAndConfirm(app, true)
				defer app.Destroy()

				cmd := exec.Command("cf", "change-stack", app.Name, newStack)
				out, err := cmd.CombinedOutput()
				Expect(err).NotTo(HaveOccurred(), string(out))
				Expect(string(out)).To(ContainSubstring("%s failed to stage on: %s. Restaging on existing stack: %s", app.Name, newStack, oldStack))

				// need to do this because change-stack execution completes while the app is still starting up, otherwise there's a 404
				Eventually(func() (string, error) { return app.GetBody("/") }, 3*time.Minute).Should(ContainSubstring(appBody))
			})
		})
	})

	when.Pend("Audit Stack", func() {
		//const appCount = cf.V3ResultsPerPage + 1 TODO:// Fix this to test multi-page results
		const appCount = 10
		var (
			apps               [appCount]*cutlass.App
			spaceName, orgName string
			err                error
			stacks             = []string{oldStack, newStack}
		)

		it.Before(func() {
			Expect(CreateStack(oldStack, oldStackDescription)).To(Succeed())
			Expect(CreateStack(newStack, newStackDescription)).To(Succeed())
			orgName, spaceName, err = GetOrgAndSpace()
			Expect(err).ToNot(HaveOccurred())

			wg := sync.WaitGroup{}
			wg.Add(appCount)
			for i := 0; i < appCount; i++ {
				apps[i] = cutlass.New(filepath.Join("testdata", "simple_app"))
				apps[i].Stack = stacks[i%len(stacks)]
				apps[i].Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				apps[i].Memory = memory
				apps[i].Disk = disk

				go func(i int) { // Maybe use a worker pool to not bombard our api
					defer wg.Done()
					PushAppAndConfirm(apps[i], true)
				}(i)
			}
			wg.Wait()
		})

		it.After(func() {
			for _, app := range apps {
				Expect(app.Destroy()).To(Succeed())
			}
			cmd := exec.Command("cf", "delete-orphaned-routes", "-f")
			Expect(cmd.Run()).To(Succeed())
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

	when("Delete Stack", func() {
		it.Before(func() {
			Expect(CreateStack(fakeStack, oldStackDescription)).To(Succeed())
			Expect(CreateStack(oldStack, oldStackDescription)).To(Succeed())
			Expect(CreateStack(newStack, newStackDescription)).To(Succeed())
		})

		it("should delete the stack", func() {
			cmd := exec.Command("cf", "delete-stack", fakeStack, "-f")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(string(output)).To(ContainSubstring(fmt.Sprintf("%s has been deleted", fakeStack)))
		})

		when("an app is using the stack", func() {
			var (
				app *cutlass.App
			)

			it.Before(func() {
				app = cutlass.New(filepath.Join("testdata", "simple_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
			})

			it.After(func() {
				if app != nil {
					Expect(app.Destroy()).To(Succeed())
				}
				app = nil
			})

			it("fails to delete the stack", func() {
				PushAppAndConfirm(app, true)
				cmd := exec.Command("cf", "delete-stack", oldStack, "-f")
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(out)).To(ContainSubstring("failed to delete stack " + oldStack))
			})
		})

		when("a buildpack is using the stack", func() {
			it.Before(func() {
				Expect(CreateBuildpack(fakeBuildpack, fakeStack)).To(Succeed())
			})
			it.After(func() {
				cmd := exec.Command("cf", "delete-buildpack", fakeBuildpack, "-f")
				Expect(cmd.Run()).To(Succeed())
				cmd = exec.Command("cf", "delete-stack", fakeStack, "-f")
				Expect(cmd.Run()).To(Succeed())
			})
			it("fails to delete the stack", func() {
				cmd := exec.Command("cf", "delete-stack", fakeStack, "-f")
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(out)).To(ContainSubstring("you still have buildpacks associated to " + fakeStack))
			})
		})
	})

}

func PushAppAndConfirm(app *cutlass.App, start bool) {
	Expect(app.Push()).To(Succeed(), fmt.Sprintf("App: %v", app))
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

	if !start {
		cmd := exec.Command("cf", "stop", app.Name)
		Expect(cmd.Run()).To(Succeed())
	}
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

func CreateStack(stackName, description string) error {
	data := fmt.Sprintf(`{"name":"%s", "description":"%s"}`, stackName, description)
	cmd := exec.Command("cf", "curl", "/v2/stacks", "-X", "POST", "-d", data)

	return cmd.Run()
}

func CreateBuildpack(buildpackName, stackName string) error {
	data := fmt.Sprintf(`{"name":"%s", "stack":"%s"}`, buildpackName, stackName)
	cmd := exec.Command("cf", "curl", "/v2/buildpacks", "-X", "POST", "-d", data)

	return cmd.Run()
}
