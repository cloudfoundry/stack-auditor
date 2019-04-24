package resources

type ErrorsJson struct {
	Errors []struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
		Code   int    `json:"code"`
	} `json:"errors"`
}
