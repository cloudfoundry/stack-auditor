package resources

type SpacesJSON struct {
	TotalResults int    `json:"total_results"`
	TotalPages   int    `json:"total_pages"`
	PrevURL      string `json:"prev_url"`
	NextURL      string `json:"next_url"`
	Resources    []struct {
		Metadata struct {
			GUID string `json:"guid"`
			URL  string `json:"url"`
		} `json:"metadata"`
		Entity struct {
			Name             string `json:"name"`
			OrganizationGUID string `json:"organization_guid"`
		} `json:"entity"`
	} `json:"resources"`
}

type Spaces []SpacesJSON

func (s Spaces) MakeSpaceOrgAndNameMap() (map[string]string, map[string]string) {
	spaceOrgMap := make(map[string]string)
	spaceNameMap := make(map[string]string)
	for _, spaces := range s {
		for _, space := range spaces.Resources {
			spaceNameMap[space.Metadata.GUID] = space.Entity.Name
			spaceOrgMap[space.Metadata.GUID] = space.Entity.OrganizationGUID
		}
	}
	return spaceNameMap, spaceOrgMap
}
