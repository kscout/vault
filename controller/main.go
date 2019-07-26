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
	"encoding/base64"
	"encoding/json"
	"os"
	"os/signal"

	"github.com/kscout/vault/controller/config"

	"github.com/Noah-Huppert/golog"
	vaultAPI "github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	kubeErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kubeRest "k8s.io/client-go/rest"
)

func main() {
	// {{{1 Initialize controller
	ctx, cancelCtx := context.WithCancel(context.Background())

	logger := golog.NewStdLogger("controller")

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

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatalf("failed to load configuration: %s",
			err.Error())
	}

	vault, err := vaultAPI.NewClient(&vaultAPI.Config{
		Address: cfg.Vault.Addr,
	})
	if err != nil {
		logger.Fatalf("failed to create Vault API client: %s",
			err.Error())
	}
	vaultSys := vault.Sys()

	kubeCfg, err := kubeRest.InClusterConfig()
	if err != nil {
		logger.Fatalf("failed to create Kubernetes configuration: %s",
			err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		logger.Fatalf("failed to create Kubernetes client: %s",
			err.Error())
	}

	kubeSecrets := kubeClient.CoreV1().Secrets("")

	// {{{1 Initialize Vault if needed
	// {{{2 Check if Vault is initialized
	isVaultInit, err := vaultSys.InitStatus()
	if err != nil {
		logger.Fatalf("failed to get Vault init status: %s",
			err.Error())
	}

	// {{{2 Initialize Vault
	if !isVaultInit {
		// {{{3 Make initialize API request
		logger.Debug("initializing Vault")

		initResp, err := vaultSys.Init(&vaultAPI.InitRequest{
			SecretShares:    int(cfg.Init.NumKeys),
			SecretThreshold: int(cfg.Init.NumKeys),
		})
		if err != nil {
			logger.Fatalf("failed to initialize Vault: %s",
				err.Error())
		}

		logger.Debug("initialized vault")

		// {{{3 Store init credentials in Kubernetes secret
		logger.Debug("saving vault credentials")

		// {{{4 Build secret
		// {{{5 Master keys to b64 json
		keysJSON, err := json.Marshal(initResp.Keys)
		if err != nil {
			logger.Fatalf("failed to marshal vault master keys "+
				"array into JSON: %s", err.Error())
		}

		keysJSONB64 := []byte{}
		base64.StdEncoding.Encode(keysJSONB64, keysJSON)

		// {{{5 Root token b64
		rootTokenB64 := []byte{}
		base64.StdEncoding.Encode(rootTokenB64, []byte(initResp.RootToken))

		// {{{5 Secret
		secretSpec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   cfg.Init.CredsKubeSecret.Name,
				Labels: cfg.Init.CredsKubeSecret.Labels,
			},
			Type: "Opaque",
			Data: map[string][]byte{
				"Keys":      keysJSONB64,
				"RootToken": rootTokenB64,
			},
		}

		// {{{4 Check if secret exists
		_, err = kubeSecrets.Get(cfg.Init.CredsKubeSecret.Name,
			metav1.GetOptions{})

		if kubeErrors.IsNotFound(err) { // Secret not found, create
			_, err := kubeSecrets.Create(secretSpec)
			if err != nil {
				logger.Fatalf("failed to create credentials "+
					"secret \"%s\": %s",
					cfg.Init.CredsKubeSecret.Name, err.Error())
			}
		} else if err != nil { // Error getting secret
			logger.Fatalf("failed get credentials secret: %s",
				err.Error())
		} else { // Found secret, exit
			logger.Fatalf("found existing credentials secret "+
				"\"%s\" during init, cannot exist",
				cfg.Init.CredsKubeSecret.Name)
		}

		logger.Debug("saved vault credentails")
	} else {
		logger.Debug("vault already initialized")
	}
}
