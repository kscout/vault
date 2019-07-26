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
	"os"
	"os/signal"

	"github.com/kscout/vault/controller/config"

	"github.com/Noah-Huppert/golog"
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
	vault, err := vaultAPI.NewClient(&vaultAPI.Config{
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
	isVaultInit, err := vault.Sys().InitStatus()
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
			logger.Fatalf("the Kubernetes secret \"%s\" where vault credentails will be "+
				"stored already exists. The initialization process cannot continue "+
				"as we will not overwrite this secret and we need a place to store the "+
				"vault credentials", cfg.Init.CredsKubeSecret.Name)
		}

		// {{{3 Make initialize API request
		logger.Info("initializing Vault")

		initResp, err := vault.Sys().Init(&vaultAPI.InitRequest{
			SecretShares:    int(cfg.Init.NumKeys),
			SecretThreshold: int(cfg.Init.NumKeys),
		})
		if err != nil {
			logger.Fatalf("failed to initialize Vault: %s",
				err.Error())
		}

		logger.Info("initialized vault")
		logger.Info("if the controller crashes before vault credentails are saved the vault will not " +
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

		logger.Info("saved vault credentails")
		logger.Infof("vault is now safely initialized, find credentails in the \"%s\" Kubernetes "+
			"secret in the \"%s\" namespace", cfg.Init.CredsKubeSecret.Name,
			cfg.Init.CredsKubeSecret.Namespace)
	} else {
		logger.Info("vault already initialized")
	}

	// {{{1 Unseal Vault if needed
	sealStatus, err := vault.Sys().SealStatus()
	if err != nil {
		logger.Fatalf("failed to get Vault seal status: %s",
			err.Error())
	}

	if sealStatus.Sealed { // Unseal
		logger.Info("unsealing vault")

		// {{{2 Get vault credentails
		// {{{3 Get secret
		kSecrets := kClient.CoreV1().Secrets(cfg.Init.CredsKubeSecret.Namespace)

		secret, err := kSecrets.Get(cfg.Init.CredsKubeSecret.Name,
			kMetaV1.GetOptions{})
		if err != nil {
			logger.Fatalf("failed to get Vault credentails secret \"%s\": %s",
				cfg.Init.CredsKubeSecret.Name, err.Error())
		}

		// {{{3 Unmarshal master keys
		var keys []string
		if err := json.Unmarshal(secret.Data["Keys"], &keys); err != nil {
			logger.Fatalf("failed to unmarshal Vault master keys array as JSON: %s",
				err.Error())
		}

		// {{{2 Unseal
		if _, err := vault.Sys().UnsealWithOptions(&vaultAPI.UnsealOpts{Reset: true}); err != nil {
			logger.Fatalf("failed to reset unseal process before providing unseal keys: %s",
				err.Error())
		}

		for i := range keys {
			resp, err := vault.Sys().Unseal(keys[i])
			if err != nil {
				logger.Fatalf("failed to pass unseal key (index: %d): %s", err.Error())
			}

			if i == len(keys)-1 && resp.Sealed {
				logger.Fatalf("vault still sealed after last unseal key provided")
			}
		}

		logger.Info("vault unsealed")
	} else {
		logger.Info("vault already unsealed")
	}
}
