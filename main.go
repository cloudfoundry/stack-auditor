package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.cloudfoundry.org/cli/plugin"
	"code.cloudfoundry.org/cli/plugin/models"
)

type Plugin struct{}

func main() {
	plugin.Start(new(Plugin))
}

func (c *Plugin) Run(cliConnection plugin.CliConnection, args []string) {
	if len(args) == 0 {
		err := errors.New("no command line arguments provided")
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Ensure that we called the command audit-stack
	switch args[0] {
	case "audit-stack":
		info, err := Audit(cliConnection)
		if err != nil {
			log.Fatalf("error talking to cf: %v\n", err)
		}

		fmt.Println(info)

		exitChan := make(chan struct{})
		signalChan := make(chan os.Signal, 1)

		signal.Notify(make(chan os.Signal), syscall.SIGHUP)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-signalChan
			close(exitChan)
		}()

		timer := time.NewTimer(10 * time.Second)

		select {
		case <-timer.C:
			fmt.Println("10 seconds elapsed")
		case <-exitChan:
			os.Exit(128)
		}
	case "CLI-MESSAGE-UNINSTALL":
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "Unknown argument provided")
		os.Exit(17)
	}
}

func (c *Plugin) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "StackAuditor",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "audit-stack",
				HelpText: "Audit stack command's help text",

				// UsageDetails is optional
				// It is used to show help of usage of each command
				UsageDetails: plugin.Usage{
					Usage: "audit-stack\n   cf audit-stack",
				},
			},
		},
	}
}

type GuidName map[string]string

func Audit(cliConnection plugin.CliConnection) (string, error) {
	orgs, err := cliConnection.GetOrgs()
	if err != nil {
		return "", err
	}

	orgMap := makeOrgMap(orgs)
	fmt.Println("orgMap: ", orgMap)

	spaceJSON, _ := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/spaces")
	stackJSON, _ := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/stacks")
	appJSON, _ := cliConnection.CliCommandWithoutTerminalOutput("curl", "/v2/apps")

	fmt.Printf("%v \n\n", appJSON[0])
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%v \n\n", spaceJSON[0])
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%v \n\n", stackJSON[0])
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%d \n\n", len(spaceJSON))

	var spaces Spaces
	var stacks Stacks
	var apps Apps

	if err := json.Unmarshal([]byte(spaceJSON[0]), &spaces); err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(stackJSON[0]), &stacks); err != nil {
		return "", err
	}
	if err := json.Unmarshal([]byte(appJSON[0]), &apps); err != nil {
		return "", err
	}

	fmt.Println("Unmarshal Spaces: ", spaces)
	fmt.Println("Unmarshal Stacks: ", stacks)
	fmt.Println("Unmarshal Apps: ", apps)

	return "", nil
}

func makeOrgMap(orgs []plugin_models.GetOrgs_Model) map[string]string {
	m := make(map[string]string)

	for _, org := range orgs {
		m[org.Guid] = org.Name
	}
	return m
}

type Spaces struct {
	TotalResults int         `json:"total_results"`
	TotalPages   int         `json:"total_pages"`
	PrevURL      interface{} `json:"prev_url"`
	NextURL      interface{} `json:"next_url"`
	Resources    []struct {
		Metadata struct {
			GUID      string    `json:"guid"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"metadata"`
		Entity struct {
			Name                     string      `json:"name"`
			OrganizationGUID         string      `json:"organization_guid"`
			SpaceQuotaDefinitionGUID interface{} `json:"space_quota_definition_guid"`
			IsolationSegmentGUID     interface{} `json:"isolation_segment_guid"`
			AllowSSH                 bool        `json:"allow_ssh"`
			OrganizationURL          string      `json:"organization_url"`
			DevelopersURL            string      `json:"developers_url"`
			ManagersURL              string      `json:"managers_url"`
			AuditorsURL              string      `json:"auditors_url"`
			AppsURL                  string      `json:"apps_url"`
			RoutesURL                string      `json:"routes_url"`
			DomainsURL               string      `json:"domains_url"`
			ServiceInstancesURL      string      `json:"service_instances_url"`
			AppEventsURL             string      `json:"app_events_url"`
			EventsURL                string      `json:"events_url"`
			SecurityGroupsURL        string      `json:"security_groups_url"`
			StagingSecurityGroupsURL string      `json:"staging_security_groups_url"`
		} `json:"entity"`
	} `json:"resources"`
}

type Stacks struct {
	TotalResults int         `json:"total_results"`
	TotalPages   int         `json:"total_pages"`
	PrevURL      interface{} `json:"prev_url"`
	NextURL      interface{} `json:"next_url"`
	Resources    []struct {
		Metadata struct {
			GUID      string    `json:"guid"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"metadata"`
		Entity struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"entity"`
	} `json:"resources"`
}

type Apps struct {
	TotalResults int         `json:"total_results"`
	TotalPages   int         `json:"total_pages"`
	PrevURL      interface{} `json:"prev_url"`
	NextURL      interface{} `json:"next_url"`
	Resources    []struct {
		Metadata struct {
			GUID      string    `json:"guid"`
			URL       string    `json:"url"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"metadata"`
		Entity struct {
			Name                  string `json:"name"`
			Production            bool   `json:"production"`
			SpaceGUID             string `json:"space_guid"`
			StackGUID             string `json:"stack_guid"`
			Buildpack             string `json:"buildpack"`
			DetectedBuildpack     string `json:"detected_buildpack"`
			DetectedBuildpackGUID string `json:"detected_buildpack_guid"`
			EnvironmentJSON       struct {
			} `json:"environment_json"`
			Memory                   int         `json:"memory"`
			Instances                int         `json:"instances"`
			DiskQuota                int         `json:"disk_quota"`
			State                    string      `json:"state"`
			Version                  string      `json:"version"`
			Command                  interface{} `json:"command"`
			Console                  bool        `json:"console"`
			Debug                    interface{} `json:"debug"`
			StagingTaskID            string      `json:"staging_task_id"`
			PackageState             string      `json:"package_state"`
			HealthCheckType          string      `json:"health_check_type"`
			HealthCheckTimeout       interface{} `json:"health_check_timeout"`
			HealthCheckHTTPEndpoint  string      `json:"health_check_http_endpoint"`
			StagingFailedReason      interface{} `json:"staging_failed_reason"`
			StagingFailedDescription interface{} `json:"staging_failed_description"`
			Diego                    bool        `json:"diego"`
			DockerImage              interface{} `json:"docker_image"`
			DockerCredentials        struct {
				Username interface{} `json:"username"`
				Password interface{} `json:"password"`
			} `json:"docker_credentials"`
			PackageUpdatedAt     time.Time `json:"package_updated_at"`
			DetectedStartCommand string    `json:"detected_start_command"`
			EnableSSH            bool      `json:"enable_ssh"`
			Ports                []int     `json:"ports"`
			SpaceURL             string    `json:"space_url"`
			StackURL             string    `json:"stack_url"`
			RoutesURL            string    `json:"routes_url"`
			EventsURL            string    `json:"events_url"`
			ServiceBindingsURL   string    `json:"service_bindings_url"`
			RouteMappingsURL     string    `json:"route_mappings_url"`
		} `json:"entity"`
	} `json:"resources"`
}
