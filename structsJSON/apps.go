package structsJSON

import "time"

type Apps struct {
	TotalResults int    `json:"total_results"`
	TotalPages   int    `json:"total_pages"`
	PrevURL      string `json:"prev_url"`
	NextURL      string `json:"next_url"`
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
