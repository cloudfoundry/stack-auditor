package resources

// Partial structure of JSON when hitting the /v2/apps endpoint
type V2AppsJSON struct {
	NextURL string  `json:"next_url"`
	Apps    []V2App `json:"resources"`
}

type V2App struct {
	Metadata struct {
		GUID string `json:"guid"`
	} `json:"metadata"`
	Entity struct {
		Name      string `json:"name"`
		SpaceGUID string `json:"space_guid"`
		StackGUID string `json:"stack_guid"`
		State     string `json:"state"`
	} `json:"entity"`
}

// Partial structure of JSON when hitting the /v3/apps endpoint
type V3AppsJSON struct {
	Pagination struct {
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"pagination"`
	Apps []V3App `json:"resources"`
}

type V3App struct {
	GUID      string `json:"guid"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Lifecycle struct {
		Data struct {
			Stack string `json:"stack"`
		} `json:"data"`
	} `json:"lifecycle"`
	Relationships struct {
		Space struct {
			Data struct {
				GUID string `json:"guid"`
			} `json:"data"`
		} `json:"space"`
	} `json:"relationships"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
		Packages struct {
			Href string `json:"href"`
		} `json:"packages"`
		CurrentDroplet struct {
			Href string `json:"href"`
		} `json:"current_droplet"`
		Droplets struct {
			Href string `json:"href"`
		} `json:"droplets"`
		Tasks struct {
			Href string `json:"href"`
		} `json:"tasks"`
		Start struct {
			Href   string `json:"href"`
			Method string `json:"method"`
		} `json:"start"`
		Stop struct {
			Href   string `json:"href"`
			Method string `json:"method"`
		} `json:"stop"`
		Revisions struct {
			Href string `json:"href"`
		} `json:"revisions"`
		DeployedRevisions struct {
			Href string `json:"href"`
		} `json:"deployed_revisions"`
	} `json:"links"`
}
