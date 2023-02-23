package auditor_test

import (
	"encoding/json"
	"fmt"

	"github.com/cloudfoundry/stack-auditor/resources"

	"github.com/cloudfoundry/stack-auditor/auditor"
	"github.com/cloudfoundry/stack-auditor/cf"
	"github.com/cloudfoundry/stack-auditor/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	OrgName    = "commonOrg"
	SpaceName  = "commonSpace"
	AppAName   = "appA"
	AppBName   = "appB"
	AppAPath   = OrgName + "/" + SpaceName + "/" + AppAName
	AppBPath   = OrgName + "/" + SpaceName + "/" + AppBName
	StackAName = "stackA"
	StackBName = "stackB"
	AppAState  = "started"
	AppBState  = "stopped"
)

var _ = Describe("Auditor", func() {
	var (
		mockCtrl       *gomock.Controller
		mockConnection *mocks.MockCliConnection
		a              auditor.Auditor
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())

		mockConnection = mocks.SetupMockCliConnection(mockCtrl)

		a = auditor.Auditor{
			CF: cf.CF{
				Conn: mockConnection,
			},
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	When("running audit-stack", func() {
		It("Verify that cf returns the correct stack associations", func() {
			result, err := a.Audit()
			Expect(err).NotTo(HaveOccurred())

			expectedResult := AppAPath + " " + StackAName + " " + AppAState + "\n" +
				AppBPath + " " + StackBName + " " + AppBState + "\n"
			Expect(result).To(Equal(expectedResult))
		})

		It("Outputs json format when the used provides the --json flag", func() {
			a.OutputType = auditor.JSONFlag
			result, err := a.Audit()
			Expect(err).NotTo(HaveOccurred())

			var apps resources.Apps
			apps = append(apps, resources.App{
				Name:  AppAName,
				Stack: StackAName,
				Org:   OrgName,
				Space: SpaceName,
				State: AppAState,
			},
				resources.App{
					Name:  AppBName,
					Stack: StackBName,
					Org:   OrgName,
					Space: SpaceName,
					State: AppBState,
				})

			expectedResult, err := json.Marshal(&apps)
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(string(expectedResult)))
		})

		It("Outputs csv format when the used provides the --csv flag", func() {
			a.OutputType = auditor.CSVFlag
			result, err := a.Audit()
			Expect(err).NotTo(HaveOccurred())

			csvFmt := "%s,%s,%s,%s,%s\n"
			csvResult := `org,space,name,stack,state
` + fmt.Sprintf(csvFmt, OrgName, SpaceName, AppAName, StackAName, AppAState) +
				fmt.Sprintf(csvFmt, OrgName, SpaceName, AppBName, StackBName, AppBState)

			Expect(result).To(Equal(csvResult))
		})
	})
})
