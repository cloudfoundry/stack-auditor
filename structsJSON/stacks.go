package structsJSON

import "time"

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

func (s *Stacks) MakeStackMap() map[string]string {
	m := make(map[string]string)

	for _, stack := range s.Resources {
		m[stack.Metadata.GUID] = stack.Entity.Name
	}
	return m
}
