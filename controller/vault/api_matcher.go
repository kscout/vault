package vault

import (
	"context"
	"fmt"

	"github.com/Noah-Huppert/golog"
	"github.com/google/go-cmp/cmp"
	vAPI "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
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
func (m MatchAPIAction) Do(ctx context.Context, vClient *vAPI.Client) (bool, error) {
	// Make request
	var actual interface{}
	if err := m.Req.Do(ctx, vClient, &actual); err != nil {
		return false, fmt.Errorf("failed to make API request: %s", err.Error())
	}

	// Match against desired state
	return m.StateMatcher.Match(m.DesiredState, actual)
}

// APIStateMatcher compares a desired an actual state and returns if
// they match
type APIStateMatcher interface {
	Match(desired, actual interface{}) (bool, error)
}

// APIRespDataMatcher compares any desired field against the .Data field
// of an APIResp.
type APIRespDataMatcher struct {
	// Logger outputs diffs between actual and desired
	Logger golog.Logger
}

// Match compares the .Data field of the actual APIResp to any typed desired
func (m APIRespDataMatcher) Match(desired, actual interface{}) (bool, error) {
	// Cast actual to APIResp
	var actualResp APIResp
	if err := mapstructure.Decode(actual, &actualResp); err != nil {
		return false, fmt.Errorf("cannot decode actual to vault.APIResp")
	}

	// Compare
	if !cmp.Equal(desired, actualResp) {
		m.Logger.Debug(cmp.Diff(desired, actualResp))
		return false, nil
	}
	return true, nil
}
