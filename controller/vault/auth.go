package vault

// AuthMethodTune is the tuning state of a Vault auth method
//
// Vault API: https://www.vaultproject.io/api/system/auth.html#parameters-3
type AuthMethodTune struct {
	// DefaultLeaseTTL is the default time to live
	DefaultLeaseTTL int `json:"default_lease_ttl"`

	// MaxLeaseTTL is the max time to live
	MaxLeaseTTL int `json:"max_lease_ttl"`

	// Description of auth method
	Description string `json:"description"`

	// AuditNonHMACRequestKeys values of keys which audit devices will
	// not HMAC
	AuditNonHMACRequestKeys []string `json:"audit_non_hmac_request_keys"`

	// AuditNonHMACResponseKeys values of keys which audit devices will
	// not HMAC
	AuditNonHMACResponseKeys []string `json:"audit_non_hmac_response_keys"`

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

	// PassthroughRequestHeaders is a commas seperated list of headers to
	// pass through to the auth method plugin
	PassthroughRequestHeaders string `json:"passthrough_request_headers"`

	// AllowedResponseHeaders is a comma seperated list of headers to let
	// (whitelist) this auth method include in responses
	AllowedResponseHeaders string `json:"allowed_response_headers"`

	// TokenType specifies the types of token that should be returned by
	// this auth method mount. See Vault API docs for allowed values.
	TokenType string `json:"token_type"`
}
