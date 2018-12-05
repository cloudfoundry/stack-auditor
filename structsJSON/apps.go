package structsJSON

type Apps struct {
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
			Name      string `json:"name"`
			SpaceGUID string `json:"space_guid"`
			StackGUID string `json:"stack_guid"`
		} `json:"entity"`
	} `json:"resources"`
}
