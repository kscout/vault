package vault

// APIResp is the structure a response usually follows
type APIResp struct {
	// Data
	Data interface{} `json:"data"`
}
