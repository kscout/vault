/*
Controller configuration.

Pass configuration via a TOML file in the PWD or in /etc/vault-controller.

The Config.Vault.Addr value can be overriden with the
VAULT_CONTROLLER_VAULT_ADDR env var.
*/
package config

import (
	"fmt"
	"os"

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
	//
	// There is no `validate` tag because this tag is processed by the config
	// file library. And this field's value can be overriden by env var after
	// the config file loads. A user could not include the config file option
	// file include an env var, and get an error saying they didn't include
	// a value, if the validate field was here. Instead validation is done
	// in NewConfig()
	Addr string `default:"http://localhost:8200"`
}

// VaultInitConfig defines how the Vault initialization process
// is performed
type VaultInitConfig struct {
	// NumKeys is the number of pieces the master key should be split
	// into. The secret_shares and secret_threshold init parameters
	// will be set to this value.
	NumKeys uint64 `default:"5" validate:"required"`

	// CredsKubeSecret is configuration for the Kubernetes secret used to
	// store Vault credentials.
	CredsKubeSecret KubeSecretConfig `validate:"required"`
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
	GitHub *vault.GHAuthState
}

// NewConfig loads configuration from TOML files in the PWD
// or /etc/vault-controller directories.
//
// The Config.Vault.Addr field can be overriden with the
// VAULT_CONTROLLER_VAULT_ADDR env var. This allows for at
// runtime customization.
func NewConfig() (*Config, error) {
	// Load config from file
	var cfg Config
	loader := goconf.NewDefaultLoader()

	loader.AddConfigPath("*.toml")
	loader.AddConfigPath("/etc/vault-controller/*.toml")

	if err := loader.Load(&cfg); err != nil {
		return nil, fmt.Errorf("failed to load TOML configuration "+
			"file(s): %s", err.Error())
	}

	// Load some fields from env
	if vaultAddr := os.Getenv("VAULT_CONTROLLER_VAULT_ADDR"); vaultAdrr != "" {
		cfg.Vault.Addr = vaultAddr
	}

	// Do validate of fields overriden by env
	if cfg.Value.Addr == "" {
		return nil, fmt.Errorf("Vault.Addr field must be present")
	}

	return &cfg, nil
}
