/*
Operates Vault server.

Performs the following steps in order.

1. Initialize Vault. Master key parts are saved in a
Kubernetes secret under the key `keys`.

2. Unseal Vault

3. Configure using Vault API

Configured via TOML file in PWD or /etc/vault-controller.
*/
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/kscout/vault/controller/config"
	"github.com/kscout/vault/controller/vault"

	"github.com/Noah-Huppert/golog"
	"github.com/google/go-cmp/cmp"
	vaultAPI "github.com/hashicorp/vault/api"
	kCoreV1 "k8s.io/api/core/v1"
	kErrors "k8s.io/apimachinery/pkg/api/errors"
	kMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kRest "k8s.io/client-go/rest"
	kClientcmd "k8s.io/client-go/tools/clientcmd"
)

func main() {
	// {{{1 Initialize controller
	// {{{2 Process lifecycle
	ctx, cancelCtx := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		select {
		case <-ctx.Done():
			break
		case <-sigs:
			cancelCtx()
			break
		}
	}()

	// {{{2 Logger and config
	logger := golog.NewStdLogger("controller")

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatalf("failed to load configuration: %s",
			err.Error())
	}

	// {{{2 Flags
	kubeconfig := flag.String("kubeconfig", "", "(Optional) Use provided kubeconfig "+
		"to authenticate with Kubernetes API")
	flag.Parse()

	// {{{2 Vault API
	vaultClient, err := vaultAPI.NewClient(&vaultAPI.Config{
		Address: cfg.Vault.Addr,
	})
	if err != nil {
		logger.Fatalf("failed to create Vault API client: %s",
			err.Error())
	}

	// {{{2 Kubernetes API
	var kCfg *kRest.Config

	if len(*kubeconfig) > 0 { // Out of cluster
		kCfg, err = kClientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			logger.Fatalf("failed to create out of cluster "+
				"Kubernetes configurtion: %s", err.Error())
		}
	} else { // In cluster
		kCfg, err = kRest.InClusterConfig()
		if err != nil {
			logger.Fatalf("failed to create in cluster "+
				"Kubernetes configuration: %s", err.Error())
		}
	}

	kClient, err := kubernetes.NewForConfig(kCfg)
	if err != nil {
		logger.Fatalf("failed to create Kubernetes client: %s",
			err.Error())
	}

	// {{{1 Initialize Vault if needed
	// {{{2 Check if Vault is initialized
	isVaultInit, err := vaultClient.Sys().InitStatus()
	if err != nil {
		logger.Fatalf("failed to get Vault init status: %s",
			err.Error())
	}

	// {{{2 Initialize Vault
	if !isVaultInit {
		// {{{4 Check if vault credentials secret exists
		kSecrets := kClient.CoreV1().Secrets(cfg.Init.CredsKubeSecret.Namespace)

		_, err = kSecrets.Get(cfg.Init.CredsKubeSecret.Name,
			kMetaV1.GetOptions{})

		if err != nil && !kErrors.IsNotFound(err) {
			logger.Fatalf("failed to check if a vault credentials secret "+
				"already exists: %s", err.Error())
		} else if err == nil {
			logger.Fatalf("the Kubernetes secret \"%s\" where vault credentials will be "+
				"stored already exists. The initialization process cannot continue "+
				"as we will not overwrite this secret and we need a place to store the "+
				"vault credentials", cfg.Init.CredsKubeSecret.Name)
		}

		// {{{3 Make initialize API request
		logger.Info("initializing Vault")

		initResp, err := vaultClient.Sys().Init(&vaultAPI.InitRequest{
			SecretShares:    int(cfg.Init.NumKeys),
			SecretThreshold: int(cfg.Init.NumKeys),
		})
		if err != nil {
			logger.Fatalf("failed to initialize Vault: %s",
				err.Error())
		}

		logger.Info("initialized vault")
		logger.Info("if the controller crashes before vault credentials are saved the vault will not " +
			"be accessible ever. This is acceptable as an unititalized vault should be empty")

		// {{{3 Store init credentials in Kubernetes secret
		logger.Info("saving vault credentials")

		// {{{4 Build secret
		var keysJSON []byte
		if jsonDat, err := json.Marshal(initResp.Keys); err != nil {
			logger.Fatalf("failed to marshal vault master keys "+
				"array into JSON: %s", err.Error())
		} else {
			keysJSON = jsonDat
		}

		secretSpec := &kCoreV1.Secret{
			ObjectMeta: kMetaV1.ObjectMeta{
				Name:   cfg.Init.CredsKubeSecret.Name,
				Labels: cfg.Init.CredsKubeSecret.Labels,
			},
			Type: "Opaque",
			Data: map[string][]byte{
				"Keys": keysJSON,
			},
			StringData: map[string]string{
				"RootToken": initResp.RootToken,
			},
		}

		// {{{4 Create secret
		_, err = kSecrets.Create(secretSpec)
		if err != nil {
			logger.Fatalf("failed to create credentials "+
				"secret \"%s\": %s",
				cfg.Init.CredsKubeSecret.Name, err.Error())
		}

		logger.Info("saved vault credentials")
		logger.Infof("vault is now safely initialized, find credentials in the \"%s\" Kubernetes "+
			"secret in the \"%s\" namespace", cfg.Init.CredsKubeSecret.Name,
			cfg.Init.CredsKubeSecret.Namespace)
	} else {
		logger.Info("vault already initialized")
	}

	// {{{1 Get vault credentials
	// {{{2 Get secret
	kSecrets := kClient.CoreV1().Secrets(cfg.Init.CredsKubeSecret.Namespace)

	vaultCredsSecret, err := kSecrets.Get(cfg.Init.CredsKubeSecret.Name,
		kMetaV1.GetOptions{})
	if err != nil {
		logger.Fatalf("failed to get Vault credentials secret \"%s\": %s",
			cfg.Init.CredsKubeSecret.Name, err.Error())
	}

	// {{{2 Unmarshal master keys in array
	var vaultMasterKeys []string
	if err := json.Unmarshal(vaultCredsSecret.Data["Keys"], &vaultMasterKeys); err != nil {
		logger.Fatalf("failed to unmarshal Vault master keys array as JSON: %s",
			err.Error())
	}

	// {{{2 Give vault client root token
	vaultClient.SetToken(string(vaultCredsSecret.Data["RootToken"]))

	// {{{1 Unseal Vault if needed
	sealStatus, err := vaultClient.Sys().SealStatus()
	if err != nil {
		logger.Fatalf("failed to get Vault seal status: %s",
			err.Error())
	}

	if sealStatus.Sealed { // Unseal
		logger.Info("unsealing vault")

		if _, err := vaultClient.Sys().UnsealWithOptions(&vaultAPI.UnsealOpts{Reset: true}); err != nil {
			logger.Fatalf("failed to reset unseal process before providing unseal keys: %s",
				err.Error())
		}

		for i, key := range vaultMasterKeys {
			resp, err := vaultClient.Sys().Unseal(key)
			if err != nil {
				logger.Fatalf("failed to pass unseal key (index: %d): %s", err.Error())
			}

			if i == len(vaultMasterKeys)-1 && resp.Sealed {
				logger.Fatalf("vault still sealed after last unseal key provided")
			}
		}

		logger.Info("vault unsealed")
	} else {
		logger.Info("vault already unsealed")
	}

	// {{{1 Setup Vault authentication
	if cfg.Auth.GitHub != nil {
		// {{{1 Enable Vault GitHub authentication if required
		vaultAuths, err := vaultClient.Sys().ListAuth()
		if err != nil {
			logger.Fatalf("failed to list Vault authentication providers: %s",
				err.Error())
		}
		if _, ok := vaultAuths["github/"]; !ok { // Not found create
			logger.Info("enabling GitHub Vault authentication")

			req := vaultClient.NewRequest("POST", "/v1/sys/auth/github")
			err = req.SetJSONBody(map[string]interface{}{
				"description": fmt.Sprintf("Allows GitHub users in the %s organization to authenticate",
					cfg.Auth.GitHub.Organization),
				"type": "github",
			})
			if err != nil {
				logger.Fatalf("failed to encode enable Vault GitHub authentication request body to JSON: %s",
					err.Error())
			}

			_, err := vaultClient.RawRequestWithContext(ctx, req)
			if err != nil {
				logger.Fatalf("failed to enable Vault GitHub authentication: %s", err.Error())
			}

			logger.Info("enabled GitHub Vault authentication")
		} else {
			logger.Info("vault GitHub authentication already enabled")
		}

		// {{{1 Configure Vault GitHub authentication if required
		getReq := vaultClient.NewRequest("GET", "/v1/auth/github/config")
		getResp, err := vaultClient.RawRequestWithContext(ctx, getReq)
		if err != nil {
			logger.Fatalf("failed to get Vault GitHub authentication configuration: %s",
				err.Error())
		}

		var getRespDat vault.GetGHAuthResp
		if err := getResp.DecodeJSON(&getRespDat); err != nil {
			logger.Fatalf("failed to decode get Vault GitHub authentication configuration "+
				"response as JSON: %s", err.Error())
		}

		if !cmp.Equal(getRespDat.Data, *cfg.Auth.GitHub) { // Configuration is different, set
			logger.Info("configuring Vault GitHub authentication")

			setReq := vaultClient.NewRequest("POST", "/v1/auth/github/config")
			if err := setReq.SetJSONBody(*cfg.Auth.GitHub); err != nil {
				logger.Fatalf("failed to encode desired Vault GitHub authentication "+
					"as JSON: %s", err.Error())
			}

			if _, err := vaultClient.RawRequestWithContext(ctx, setReq); err != nil {
				logger.Fatalf("failed to call set Vault GitHub authentication endpoint: %s",
					err.Error())
			}

			logger.Info("configured Vault GitHub authentication")
		} else {
			logger.Info("vault GitHub authentication already configured")
		}
	}

	/*
		for _, endpoint := range cfg.DesiredEndpoints {
			logger.Infof("setting \"%s\" Vault API endpoint state", endpoint.Path)

			// {{{2 Get actual state
			getReq := vault.NewRequest("GET", endpoint.Path)
			getResp, err := vault.RawRequestWithContext(ctx, getReq)
			if err != nil {
				logger.Fatalf("failed to get Vault API endpoint state  \"%s\": %s",
					endpoint.Path, err.Error())
			}

			var actual interface{}
			if err := getResp.DecodeJSON(&actual); err != nil {
				logger.Fatalf("failed to decode vault API get endpoint state response \"%s\": %s",
					endpoint.Path, err.Error())
			}

			logger.Debugf("actual=%#v, expected=%#v", actual, endpoint.State)
		}
	*/
}
