package vault

// GHAuthState is the state of a Vault GitHub authentication provider
type GHAuthState struct {
	// Organization is the GitHub organization a user must be part of to
	// be authenticated
	Organization string `json:"organization,omitempty" validate:"required"`

	// BaseURL is the GitHub API base URL, defaults to public GitHub
	BaseURL string `json:"base_url,omitempty"`

	// TTL is how login tokens from this provider will last
	TTL string `json:"ttl,omitempty"`

	// MaxTTL is the maximimum duration after which authentication will expire
	MaxTTL string `json:"max_ttl,omitempty"`
}
