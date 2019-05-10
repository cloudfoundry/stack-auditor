package resources

import "time"

type DropletListJSON struct {
	Pagination struct {
		TotalResults int `json:"total_results"`
		TotalPages   int `json:"total_pages"`
		First        struct {
			Href string `json:"href"`
		} `json:"first"`
		Last struct {
			Href string `json:"href"`
		} `json:"last"`
		Next     interface{} `json:"next"`
		Previous interface{} `json:"previous"`
	} `json:"pagination"`
	Resources []DropletJSON `json:"resources"`
}

type DropletJSON struct {
	GUID      string    `json:"guid"`
	State     string    `json:"state"`
	Stack     string    `json:"stack"`
	CreatedAt time.Time `json:"created_at"`
}
