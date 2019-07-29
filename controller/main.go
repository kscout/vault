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
	"time"

	"github.com/kscout/vault/controller/config"
	"github.com/kscout/vault/controller/errors"
	"github.com/kscout/vault/controller/vault"

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

	handleErr := errors.GetErrorHandler(logger)

	cfg, err := config.NewConfig()
	handleErr(err, "failed to load configuration")

	// {{{2 Flags
	kubeconfig := flag.String("kubeconfig", "", "Use provided kubeconfig "+
		"to authenticate with Kubernetes API. Defaults to using cluster "+
		"config.")
	flag.Parse()

	// {{{2 Vault API
	vClient, err := vaultAPI.NewClient(&vaultAPI.Config{
		Address: cfg.Vault.Addr,
	})
	handleErr(err, "failed to create Vault API client")

	// {{{2 Kubernetes API
	var kCfg *kRest.Config

	if len(*kubeconfig) > 0 { // Out of cluster
		kCfg, err = kClientcmd.BuildConfigFromFlags("", *kubeconfig)
		handleErr(err, "failed to create out of cluster "+
			"Kubernetes configurtion")
	} else { // In cluster
		kCfg, err = kRest.InClusterConfig()
		handleErr(err, "failed to create in cluster "+
			"Kubernetes configuration")
	}

	kClient, err := kubernetes.NewForConfig(kCfg)
	handleErr(err, "failed to create Kubernetes client")

	// ctrlLoopTimer is used to wait before executing the control loop.
	// Starts to wait 0 seconds to make the control loop run immediately
	// when launched. From then on it will wait 15s.
	ctrlLoopTimer := time.NewTimer(0)

	runCtrlLoop := true

	for runCtrlLoop {
		select {
		case <-ctx.Done():
			logger.Info("stopping")
			runCtrlLoop = false
			break
		case <-ctrlLoopTimer.C:
			// {{{1 Initialize Vault if needed
			// {{{2 Check if Vault is initialized
			isVaultInit, err := vClient.Sys().InitStatus()
			handleErr(err, "failed to get Vault init status")

			// {{{2 Initialize Vault
			if !isVaultInit {
				// {{{4 Check if vault credentials secret exists
				kSecrets := kClient.CoreV1().
					Secrets(cfg.Init.CredsSecret.Namespace)

				_, err = kSecrets.Get(cfg.Init.CredsSecret.Name,
					kMetaV1.GetOptions{})

				if err != nil && !kErrors.IsNotFound(err) {
					logger.Fatalf("failed to check if a "+
						"vault credentials secret "+
						"already exists: %s", err.Error())
				} else if err == nil {
					logger.Fatalf("the Kubernetes secret "+
						"\"%s\" where vault credentials "+
						"will be stored already exists. "+
						"The initialization process "+
						"cannot continue as we will not "+
						"overwrite this secret and we "+
						"need a place to store the "+
						"vault credentials",
						cfg.Init.CredsSecret.Name)
				}

				// {{{3 Make initialize API request
				initResp, err := vClient.Sys().Init(&vaultAPI.InitRequest{
					SecretShares:    int(cfg.Init.NumKeys),
					SecretThreshold: int(cfg.Init.NumKeys),
				})
				handleErr(err, "failed to initialize Vault")

				logger.Info("initialized vault")
				logger.Info("if the controller crashes before " +
					"vault credentials are saved the vault " +
					"will not be accessible ever. This is " +
					"acceptable as an unititalized vault " +
					"should be empty")

				// {{{3 Store init credentials in Kubernetes secret
				// {{{4 Build secret
				keysJSON, err := json.Marshal(initResp.Keys)
				handleErr(err, "failed to marshal vault "+
					"master keys array into JSON")

				secretSpec := &kCoreV1.Secret{
					ObjectMeta: kMetaV1.ObjectMeta{
						Name:   cfg.Init.CredsSecret.Name,
						Labels: cfg.Init.CredsSecret.Labels,
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
				handleErr(err, "failed to create credentials "+
					"secret \"%s\"", cfg.Init.CredsSecret.Name)

				logger.Info("saved vault credentials")
				logger.Infof("vault is now safely initialized, "+
					"find credentials in the "+
					"\"%s\" Kubernetes secret in the "+
					"\"%s\" namespace",
					cfg.Init.CredsSecret.Name,
					cfg.Init.CredsSecret.Namespace)
			} else {
				logger.Info("vault already initialized")
			}

			// {{{1 Get vault credentials
			// {{{2 Get secret
			kSecrets := kClient.CoreV1().
				Secrets(cfg.Init.CredsSecret.Namespace)

			credsSecret, err := kSecrets.Get(cfg.Init.CredsSecret.Name,
				kMetaV1.GetOptions{})
			handleErr(err, "failed to get Vault credentials "+
				"secret \"%s\"", cfg.Init.CredsSecret.Name)

			// {{{2 Unmarshal master keys in array
			var vaultMasterKeys []string
			err = json.Unmarshal(credsSecret.Data["Keys"],
				&vaultMasterKeys)
			handleErr(err, "failed to unmarshal Vault master keys "+
				"array as JSON")

			// {{{2 Give vault client root token
			vClient.SetToken(string(credsSecret.Data["RootToken"]))

			// {{{1 Unseal Vault if needed
			sealStatus, err := vClient.Sys().SealStatus()
			handleErr(err, "failed to get Vault seal status")

			if sealStatus.Sealed { // Unseal
				_, err = vClient.Sys().UnsealWithOptions(
					&vaultAPI.UnsealOpts{Reset: true})
				handleErr(err, "failed to reset unseal process "+
					"before providing unseal keys")

				for i, key := range vaultMasterKeys {
					resp, err := vClient.Sys().Unseal(key)
					handleErr(err, "failed to pass unseal "+
						"key (index: %d)")

					if i == len(vaultMasterKeys)-1 &&
						resp.Sealed {
						logger.Fatalf("vault still " +
							"sealed after last " +
							"unseal key provided")
					}
				}

				logger.Info("vault unsealed")
			} else {
				logger.Info("vault already unsealed")
			}

			// {{{1 Setup Vault authentication
			if cfg.Auth.GitHub != nil {
				// {{{1 Enable Vault GitHub authentication if required
				vaultAuths, err := vClient.Sys().ListAuth()
				handleErr(err, "failed to list Vault "+
					"authentication providers")
				if _, ok := vaultAuths["github/"]; !ok { // Not found create
					req := vault.APIReq{
						Method: "POST",
						Path:   "/v1/sys/auth/github",
						Data: map[string]interface{}{
							"description": fmt.Sprintf("Allows GitHub "+
								"users in the %s organization "+
								"to authenticate",
								cfg.Auth.GitHub.Method.Organization),
							"type": "github",
						},
					}
					err := req.Do(ctx, vClient, nil)
					handleErr(err, "failed to enable Vault "+
						"GitHub authentication")

					logger.Info("enabled GitHub Vault authentication")
				} else {
					logger.Info("vault GitHub authentication already enabled")
				}

				// {{{2 Tune Vault GitHub authentication if required
				setTune := vault.SetAPIAction{
					Path: "/v1/sys/auth/github/tune",
					DesiredState: vault.AuthMethodTune{
						ListingVisibility: "unauth",
					},
					StateMatcher: vault.AuthMethodTuneMatcher{
						Logger: logger.GetChild("matcher"),
					},
				}

				didSetTune, err := setTune.Do(ctx, vClient)
				handleErr(err, "failed to tune Vault GitHub authentication")

				if didSetTune {
					logger.Info("tuned Vault GitHub authentication")
				} else {
					logger.Info("vault GitHub authentication already tuned")
				}

				// {{{1 Configure Vault GitHub authentication if required
				setGHCfg := vault.SetAPIAction{
					Path:         "/v1/auth/github/config",
					DesiredState: cfg.Auth.GitHub.Method,
					StateMatcher: vault.APIRespDataMatcher{
						Logger: logger.GetChild("matcher"),
					},
				}
				didSetGHCfg, err := setGHCfg.Do(ctx, vClient)
				handleErr(err, "failed to configure Vault "+
					"GitHub authentication")
				if didSetGHCfg {
					logger.Info("configured Vault GitHub authentication")
				} else {
					logger.Info("vault GitHub authentication " +
						"already configured")
				}

				// {{{1 Set GitHub team policy map if required
				matchTeamMap := vault.MatchAPIAction{
					Req: vault.APIReq{
						Method: "GET",
						Path:   "/v1/auth/github/map/teams",
					},
					DesiredState: cfg.Auth.GitHub.TeamPolicies,
					StateMatcher: vault.APIRespDataMatcher{
						Logger: logger.GetChild("matcher"),
					},
				}
				teamMapMatches, err := matchTeamMap.Do(ctx, vClient)
				handleErr(err, "failed to check if actual GitHub team "+
					"policy map matches desired state")

				if !teamMapMatches {
					for ghTeam := range cfg.Auth.GitHub.TeamPolicies {
						setTeamMapReq := vault.APIReq{
							Method: "POST",
							Path:   fmt.Sprintf("/v1/auth/github/map/teams/%s", ghTeam),
							Data: map[string]interface{}{
								"data": map[string]interface{}{
									"value": cfg.Auth.GitHub.TeamPolicies[ghTeam],
								},
							},
						}
						err = setTeamMapReq.Do(ctx, vClient, nil)
						handleErr(err, "failed to set Vault GitHub team "+
							"map for GitHub team \"%s\"", ghTeam)
					}

					logger.Info("configured GitHub team policies map")
				} else {
					logger.Info("already configured GitHub team " +
						"policies map")
				}

			}

			logger.Debug("ran control loop")

			ctrlLoopTimer.Reset(time.Second * 15)

			break
		}
	}
}
