package resources

type DropletJSON struct {
	GUID  string      `json:"guid"`
	State string      `json:"state"`
	Error interface{} `json:"error"`
}
