package structsJSON

type Stacks struct {
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
			Name string `json:"name"`
		} `json:"entity"`
	} `json:"resources"`
}

func (s *Stacks) MakeStackMap() map[string]string {
	m := make(map[string]string)

	for _, stack := range s.Resources {
		m[stack.Metadata.GUID] = stack.Entity.Name
	}
	return m
}
