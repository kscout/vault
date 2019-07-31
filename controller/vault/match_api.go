package vault

import (
	"context"
	"fmt"

	vAPI "github.com/hashicorp/vault/api"
)

// MatchAPIAction determines if the response from an API endpoint matches
type MatchAPIAction struct {
	// Req is the request which will generate the response to match against
	Req APIReq

	// DesiredState to match against
	DesiredState interface{}

	// StateMatcher compares the desired state to the actual state
	StateMatcher APIStateMatcher
}

// Do action. Returns true if response state matches desired state
func (m MatchAPIAction) Do(ctx context.Context, vClient *vAPI.Client, actual interface{}) (bool, error) {
	// Make request
	if err := m.Req.Do(ctx, vClient, &actual); err != nil {
		return false, fmt.Errorf("failed to make API request: %s", err.Error())
	}

	// Match against desired state
	return m.StateMatcher.Match(m.DesiredState, actual)
}
