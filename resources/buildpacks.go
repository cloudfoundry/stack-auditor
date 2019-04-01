package resources

type BuildpacksJSON struct {
	TotalResults int         `json:"total_results"`
	TotalPages   int         `json:"total_pages"`
	PrevURL      string      `json:"prev_url"`
	NextURL      string      `json:"next_url"`
	BuildPacks   []BuildPack `json:"resources"`
}

type BuildPack struct {
	Metadata struct {
		GUID string `json:"guid"`
		URL  string `json:"url"`
	} `json:"metadata"`
	Entity struct {
		Name     string `json:"name"`
		Stack    string `json:"stack"`
		Position int    `json:"position"`
		Enabled  bool   `json:"enabled"`
		Locked   bool   `json:"locked"`
		Filename string `json:"filename"`
	} `json:"entity"`
}
