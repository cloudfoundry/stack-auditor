package resources

type StacksJSON struct {
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

type Stacks []StacksJSON

func (s Stacks) MakeStackMap() map[string]string {
	stackMap := make(map[string]string)
	for _, stacks := range s {
		for _, stack := range stacks.Resources {
			stackMap[stack.Metadata.GUID] = stack.Entity.Name
		}
	}
	return stackMap
}
