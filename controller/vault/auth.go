package vault

import (
	"fmt"

	"github.com/Noah-Huppert/golog"
)

// AuthMethodTune is the tuning state of a Vault auth method
//
// Vault API: https://www.vaultproject.io/api/system/auth.html#parameters-3
type AuthMethodTune struct {
	// ListingVisibility specifies when this auth method will be listed.
	//
	// Valid values are:
	//
	// "unauth": If any auth method has this value then the login UI will
	//           be tweaked to show these auth methods as main tabs in the
	//           login menu. All auth methods with this value will be acessible
	//           in the "Other" menu.
	// "": Do nothing, if any other auth method specifies "unauth" then
	//     specifying "" for this auth method will make it show up in the
	//     "Other" tab in the login UI
	//
	// Docs: https://github.com/hashicorp/vault/pull/4854
	ListingVisibility string `json:"listing_visibility"`
}

// AuthMethodTuneMatcher matches a desired AuthMethodTune against an
// actual APIResp
type AuthMethodTuneMatcher struct {
	// Logger outputs diffs between expected and actual
	Logger golog.Logger
}

// Match
func (m AuthMethodTuneMatcher) Match(expected, actual interface{}) (bool, error) {
	var expectedTune AuthMethodTune
	if t, ok := expected.(AuthMethodTune); ok {
		expectedTune = t
	} else {
		return false, fmt.Errorf("expected not vault.AuthMethodTune")
	}

	var actualMap map[string]interface{}
	if v, ok := actual.(map[string]interface{}); ok {
		actualMap = v
	} else {
		return false, fmt.Errorf("actual not map[string]interface{}")
	}

	if listing, ok := actualMap["listing_visibility"]; ok {
		return listing == expectedTune.ListingVisibility, nil
	} else {
		return false, nil
	}

	return false, nil
}
