package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/cloudfoundry/stack-auditor/changer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	oldStack      = "cflinuxfs3"
	newStack      = "cflinuxfs4"
	fakeStack     = "fakeStack"
	fakeBuildpack = "fakeBuildpack"
	appBody       = "Hello World!"
	interval      = 100 * time.Millisecond
	disk          = "128M"
	memory        = "128M"
)

var _ = Describe("Integration", func() {
	When("Change Stack", func() {
		When("the app was initially started", func() {
			It("should change the stack and remain started", func() {
				app := cutlass.New(filepath.Join("testdata", "simple_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory

				PushAppAndConfirm(app, true)
				defer app.Destroy()

				breaker := make(chan bool)
				go confirmZeroDowntime(app, breaker)

				cmd := exec.Command("cf", "change-stack", app.Name, newStack)
				output, err := cmd.CombinedOutput()
				Expect(err).ToNot(HaveOccurred())
				Expect(string(output)).To(ContainSubstring(changer.RestoringStateMsg, "STARTED"))
				close(breaker)
			})
		})

		When("the app was initially stopped", func() {
			It("it should change the stack and remain stopped", func() {
				app := cutlass.New(filepath.Join("testdata", "simple_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory

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

		When("the app cannot stage on the target stack", func() {
			It("restarts itself on the old stack", func() {
				app := cutlass.New(filepath.Join("testdata", "does_not_stage_on_fs4"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/ruby-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory

				PushAppAndConfirm(app, true)
				defer app.Destroy()

				breaker := make(chan bool)
				go confirmZeroDowntime(app, breaker)

				cmd := exec.Command("cf", "change-stack", app.Name, newStack)
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred(), string(out))
				Expect(string(out)).To(ContainSubstring(changer.ErrorStaging, newStack))

				// need to do this because change-stack execution completes while the app is still starting up, otherwise there's a 404
				Eventually(func() (string, error) { return app.GetBody("/") }, 3*time.Minute).Should(ContainSubstring(appBody))
				close(breaker)
			})
		})

		When("the app cannot run on the target stack", func() {
			It("restarts itself on the old stack", func() {
				app := cutlass.New(filepath.Join("testdata", "does_not_run_on_fs4"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
				app.Disk = disk
				app.Memory = memory

				PushAppAndConfirm(app, true)
				defer app.Destroy()

				breaker := make(chan bool)
				go confirmZeroDowntime(app, breaker)

				cmd := exec.Command("cf", "change-stack", app.Name, newStack)
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred(), string(out))
				Expect(string(out)).To(ContainSubstring(changer.ErrorRestartingApp, newStack))

				// need to do this because change-stack execution completes while the app is still starting up, otherwise there's a 404
				Eventually(func() (string, error) {
					return app.GetBody("/")
				}, 3*time.Minute).Should(ContainSubstring(appBody))
				close(breaker)
			})
		})
	})

	PWhen("Audit Stack", func() {
		//const appCount = cf.V3ResultsPerPage + 1 TODO:// Fix this to test multi-page results
		const appCount = 10
		var (
			apps               [appCount]*cutlass.App
			spaceName, orgName string
			err                error
			stacks             = []string{oldStack, newStack}
		)

		BeforeEach(func() {
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

		AfterEach(func() {
			for _, app := range apps {
				Expect(app.Destroy()).To(Succeed())
			}
			cmd := exec.Command("cf", "delete-orphaned-routes", "-f")
			Expect(cmd.Run()).To(Succeed())
		})

		It("prints all apps with their orgs spaces and stacks", func() {
			cmd := exec.Command("cf", "audit-stack")
			output, err := cmd.Output()
			Expect(err).NotTo(HaveOccurred())

			for i, app := range apps {
				Expect(string(output)).To(ContainSubstring(fmt.Sprintf("%s/%s/%s %s", orgName, spaceName, app.Name, stacks[i%len(stacks)])))
			}
		})
	})

	When("Delete Stack", func() {
		BeforeEach(func() {
			Expect(CreateStack(fakeStack, "fake stack")).To(Succeed())
		})

		It("should delete the stack", func() {
			cmd := exec.Command("cf", "delete-stack", fakeStack, "-f")
			output, err := cmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))
			Expect(string(output)).To(ContainSubstring(fmt.Sprintf("%s has been deleted", fakeStack)))
		})

		When("an app is using the stack", func() {
			var (
				app *cutlass.App
			)

			BeforeEach(func() {
				app = cutlass.New(filepath.Join("testdata", "simple_app"))
				app.Buildpacks = []string{"https://github.com/cloudfoundry/binary-buildpack#master"}
				app.Stack = oldStack
			})

			AfterEach(func() {
				if app != nil {
					Expect(app.Destroy()).To(Succeed())
				}
				app = nil
			})

			It("fails to delete the stack", func() {
				PushAppAndConfirm(app, true)
				cmd := exec.Command("cf", "delete-stack", oldStack, "-f")
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(out)).To(ContainSubstring("failed to delete stack " + oldStack))
			})
		})

		When("a buildpack is using the stack", func() {
			BeforeEach(func() {
				Expect(CreateBuildpack(fakeBuildpack, fakeStack)).To(Succeed())
			})

			AfterEach(func() {
				cmd := exec.Command("cf", "delete-buildpack", fakeBuildpack, "-f")
				Expect(cmd.Run()).To(Succeed())
				cmd = exec.Command("cf", "delete-stack", fakeStack, "-f")
				Expect(cmd.Run()).To(Succeed())
			})

			It("fails to delete the stack", func() {
				cmd := exec.Command("cf", "delete-stack", fakeStack, "-f")
				out, err := cmd.CombinedOutput()
				Expect(err).To(HaveOccurred())
				Expect(string(out)).To(ContainSubstring("you still have buildpacks associated to " + fakeStack))
			})
		})
	})
})

func PushAppAndConfirm(app *cutlass.App, start bool) {
	Expect(app.Push()).To(Succeed(), fmt.Sprintf("Name: %v", app))
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

func confirmZeroDowntime(app *cutlass.App, breaker chan bool) {
	defer GinkgoRecover()
	for {
		select {
		case <-breaker:
			return
		default:
			body, err := app.GetBody("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(Equal(appBody))
			time.Sleep(interval)
		}
	}
}
