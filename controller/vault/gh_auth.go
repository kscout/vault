package vault

// GHAuthState is the state of a Vault GitHub authentication provider
type GHAuthState struct {
	// Organization is the GitHub organization a user must be part of to
	// be authenticated
	Organization string `json:"organization" validate:"required"`

	// BaseURL is the GitHub API base URL, defaults to public GitHub
	BaseURL string `json:"base_url"`

	// TTL is how login tokens from this provider will last
	TTL int `json:"ttl"`

	// MaxTTL is the maximimum duration after which authentication will expire
	MaxTTL int `json:"max_ttl"`
}
