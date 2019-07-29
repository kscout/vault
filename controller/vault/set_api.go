package vault

import (
	"context"
	"fmt"

	vAPI "github.com/hashicorp/vault/api"
)

// SetAPIAction retrieves the state of a Vault API resource and sets it
// to a desired state if neccessary.
//
// The API resource should return its state in response to get requests.
// A custom matcher determines if the desired state and actual state match.
type SetAPIAction struct {
	// Path of API resource
	Path string

	// DesiredState is the desired state of the resource
	DesiredState interface{}

	// StateMatcher compares the desired state to the actual state
	StateMatcher APIStateMatcher
}

// Do logic to set API resource. A bool is returned indicating if an update
// was made (true == update).
func (a SetAPIAction) Do(ctx context.Context, vClient *vAPI.Client) (bool, error) {
	// Check if API resource state matches
	matchAct := MatchAPIAction{
		Req: APIReq{
			Method: "GET",
			Path:   a.Path,
		},
		DesiredState: a.DesiredState,
		StateMatcher: a.StateMatcher,
	}

	if match, err := matchAct.Do(ctx, vClient); err != nil {
		return false, fmt.Errorf("failed to get actual API state and determine "+
			"if it matches: %s", err.Error())
	} else if match {
		return false, nil
	}

	// Set to desired state
	setReq := APIReq{
		Method: "POST",
		Path:   a.Path,
		Data:   a.DesiredState,
	}
	if err := setReq.Do(ctx, vClient, nil); err != nil {
		return true, fmt.Errorf("failed to set API set: %s", err.Error())
	}

	return true, nil
}
