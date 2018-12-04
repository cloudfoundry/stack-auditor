package structsJSON

import "time"

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

func (s *Spaces) MakeSpaceOrgMap() map[string]string {
	m := make(map[string]string)

	for _, space := range s.Resources {
		m[space.Metadata.GUID] = space.Entity.OrganizationGUID
	}
	return m
}

func (s *Spaces) MakeSpaceNameMap() map[string]string {
	m := make(map[string]string)

	for _, space := range s.Resources {
		m[space.Metadata.GUID] = space.Entity.Name
	}
	return m
}
