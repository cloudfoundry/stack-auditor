package resources

type V3ErrorJSON struct {
	Errors []struct {
		Detail string `json:"detail"`
		Title  string `json:"title"`
		Code   int    `json:"code"`
	} `json:"errors"`
}

type V2ErrorJSON struct {
	Description string `json:"description"`
	ErrorCode   string `json:"error_code"`
	Code        int    `json:"code"`
}
