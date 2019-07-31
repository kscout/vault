package vault

import (
	"context"
	"fmt"
	"log"

	vAPI "github.com/hashicorp/vault/api"
)

// APIReq is a Vault API request
type APIReq struct {
	// Method is the HTTP method of the request
	Method string

	// Path of API endpoint
	Path string

	// Data to encode as JSON in request body. Not for .Method == GET requests
	Data interface{}
}

// Do makes an API request.
func (r APIReq) Do(ctx context.Context, vaultClient *vAPI.Client, out interface{}) error {
	req := vaultClient.NewRequest(r.Method, r.Path)

	if r.Data != nil && r.Method != "GET" {
		if err := req.SetJSONBody(r.Data); err != nil {
			return fmt.Errorf("failed to encode request body as "+
				"JSON: %s", err.Error())
		}
	}

	resp, err := vaultClient.RawRequestWithContext(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to make request: %s", err.Error())
	}

	if out != nil {
		if err := resp.DecodeJSON(out); err != nil {
			return fmt.Errorf("failed to decode response as JSON: %s",
				err.Error())
		}
	}

	return nil
}
