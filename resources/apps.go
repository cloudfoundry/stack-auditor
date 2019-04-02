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
}
