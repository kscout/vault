package vault

import (
	"log"

	"github.com/google/go-cmp/cmp"
)

// APIStateMatcher compares a desired an actual state and returns if
// they match
type APIStateMatcher interface {
	Match(desired, actual interface{}) (bool, error)
}

// APIRespDataMatcher compares any desired field against the .Data field
// of an APIResp.
type APIRespDataMatcher struct{}

// Match compares the .Data field of the actual APIResp to any typed desired
func (m APIRespDataMatcher) Match(desired, actual interface{}) (bool, error) {
	// Compare
	if !cmp.Equal(desired, actual) {
		log.Println(cmp.Diff(desired, actual))
		return false, nil
	}
	return true, nil
}
