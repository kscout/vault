/*
Controller configuration.
*/
package config

import (
	"fmt"

	"github.com/kscout/vault/controller/vault"

	"github.com/Noah-Huppert/goconf"
)

// Config defines how the controller behaves
type Config struct {
	// Vault server configuration
	Vault VaultServerConfig `validate:"required"`

	// Init is Vault initialization process parameters
	Init VaultInitConfig `validate:"required"`

	// Auth holds Vault authentication configuration
	Auth VaultAuthConfig
}

// VaultServerConfig defines details about the Vault server to control
type VaultServerConfig struct {
	// Addr is the address of the server
	Addr string `default:"http://localhost:8200" validate:"required"`
}

// VaultInitConfig defines how the Vault initialization process
// is performed
type VaultInitConfig struct {
	// NumKeys is the number of pieces the master key should be split
	// into. The secret_shares and secret_threshold init parameters
	// will be set to this value.
	NumKeys uint64 `default:"5" validate:"required"`

	// CredsSecret is configuration for the Kubernetes secret used to
	// store Vault credentials.
	CredsSecret KubeSecretConfig `validate:"required"`
}

// KubeSecretConfig defines the parameters of the Kubernetes secret managed
// by the controller
type KubeSecretConfig struct {
	// Name is the secret's name
	Name string `validate:"required"`

	// Namespace to create secret
	Namespace string `validate:"required"`

	// Labels to add to secret resource, optional
	Labels map[string]string
}

// VaultAuthConfig defines how Vault authenticates users
type VaultAuthConfig struct {
	// GitHub authentication configuration
	GitHub *VaultGHAuthConfig
}

// VaultGHAuthConfig defines how the GitHub authentication will behave
type VaultGHAuthConfig struct {
	// Method defines the Vault auth method configuration parameters
	Method *vault.GHAuthState

	// TeamPolicies defines a policy assigned to members of a GitHub team
	//
	// Keys are GitHub team slugs. Values are Vault policies.
	TeamPolicies map[string]string
}

// NewConfig loads configuration from TOML files in the PWD
// or /etc/vault-controller directories
func NewConfig() (*Config, error) {
	loader := goconf.NewDefaultLoader()

	loader.AddConfigPath("*.toml")
	loader.AddConfigPath("/etc/vault-controller/*.toml")

	var cfg Config
	if err := loader.Load(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load TOML configuration "+
			"file(s): %s", err.Error())
	}

	return &cfg, nil
}
