/*
Ensures a Vault server's API configuration resources match a desired state.

See the config package to learn how to declare a desired state.
*/
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/kscout/vault/vault-api-controller/config"

	"github.com/Noah-Huppert/golog"
	vaultApi "github.com/hashicorp/vault/api"
)

func main() {
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

	logger := golog.NewStdLogger("vault-api-controller")

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatalf("failed to load configuration: %s", err.Error())
	}

	if cfg.VaultServer.Token == "" {
		logger.Debug("VaultServer.Token was empty, in bootstrapping mode, will wait forever")
		<-ctx.Done()
		return
	}

	vault, err := vaultApi.NewClient(&vaultApi.Config{
		Address: cfg.VaultServer.Address,
	})
	if err != nil {
		logger.Fatalf("failed to create Vault client: %s", err.Error())
	}

	vault.SetToken(cfg.VaultServer.Token)

	controlLoopSleepDur := time.Second * 15
	controlLoopTimer := time.NewTimer(0)

	for {
		select {
		case <-ctx.Done():
			return
			break
		case <-controlLoopTimer.C:
			logger.Debug("running control loop")

			for desiredStateKey := range cfg.DesiredStates {
				desiredState := cfg.DesiredStates[desiredStateKey]

				logger.Debugf("desired state=%#v", desiredState)

				actualState := config.ConfigState{
					Path:  desiredState.Path,
					State: make(map[string]interface{}),
				}

				getReq, err := vault.NewRequest("GET", desiredState.Path).ToHTTP()
				if err != nil {
					logger.Fatalf("failed to build get state request for \"%s\": %s",
						desiredState.Path, err.Error())
				}

				getResp, err := http.DefaultClient.Do(getReq)
				if err != nil {
					logger.Fatalf("failed to get actual state for \"%s\": %s",
						desiredState.Path, err.Error())
				}

				getDecoder := json.NewDecoder(getResp.Body)
				if err := getDecoder.Decode(&actualState.State); err != nil {
					logger.Fatalf("failed to decode get state request for \"%s\""+
						"as json: %s", desiredState.Path, err.Error())
				}

				logger.Debugf("actual state=%#v", actualState)

				if actualState.State != desiredState.State {
					logger.Debugf("desired and actual states are different, setting \"%s\" state",
						desiredState.Path)

					setReq, err := vault.NewRequest("POST", desiredState.Path).ToHTTP()
					if err != nil {
						logger.Fatalf("failed to build set state request for \"%s\": %s",
							desiredState.Path, err.Error())
					}

					setBuf := bytes.NewBuffer(nil)
					setEncoder := json.NewEncoder(setBuf)

					if err := setEncoder.Encode(desiredState.State); err != nil {
						logger.Fatalf("failed to encode desired state for \"%s\" "+
							"as json: %s", desiredState.Path, err.Error())
					}

					setResp, err := http.DefaultClient.Do(setReq)
					if err != nil {
						logger.Fatalf("failed to set state for \"%s\": %s",
							desiredState.Path, err.Error())
					}

					logger.Debugf("set actual \"%s\" state to desired state", desiredState.Path)
				}
			}

			logger.Debug("ran control loop, sleep for 15s")

			controlLoopTimer.Reset(controlLoopSleepDur)
			break
		}
	}
}
