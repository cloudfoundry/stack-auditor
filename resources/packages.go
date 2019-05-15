package resources

type PackagerJSON struct {
	// TOOD: do we really need paginated results?
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
	Resources []Package `json:"resources"`
}

type Package struct {
	GUID string `json:"guid"`
	Type string `json:"type"`
	Data struct {
		Error    interface{} `json:"error"`
		Checksum struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"checksum"`
	} `json:"data"`
	State string `json:"state"`
}
