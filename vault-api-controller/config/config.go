/*
TOML 4.0 configuration file which declares the desired Vault
configuration state.

Loads configuration from any *.toml file in the current working
directory or from /etc/vault-api-controller.
*/
package config

import (
	"fmt"

	"github.com/Noah-Huppert/goconf"
)

// Config
type Config struct {
	// DesiredStates holds the desired Vault configuration state
	//
	// Keys are human readable IDs
	//
	// Values are API configuration resource states
	DesiredStates map[string]ConfigState `validate:"required"`

	// VaultServer configuration
	VaultServer VaultServerConfig `validate:"required"`
}

// ConfigState is the state of a configuration resource from the Vault API
type ConfigState struct {
	// Path in Vault API which configuration resource is located
	Path string `validate:"required"`

	// State of resource
	State map[string]interface{} `validate:"required"`
}

// VaultServerConfig holds details about the Vault server
type VaultServerConfig struct {
	// Address is the address of the Vault server
	Address string `validate:"required"`

	// Token used to authenticate with the Vault server
	Token string `validate:"required"`
}

// NewConfig loads configuration from files
func NewConfig() (*Config, error) {
	loader := goconf.NewDefaultLoader()

	loader.AddConfigPath("*.toml")
	loader.AddConfigPath("/etc/vault-api-controller")

	var cfg Config
	if err := loader.Loader(&cfg); err != nil {
		return nil, fmt.Errorf("error loading configuration: %s", err.Error())
	}
	return &cfg, nil
}
