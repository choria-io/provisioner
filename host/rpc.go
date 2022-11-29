// Copyright (c) 2018-2021, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package host

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/backoff"
	"github.com/choria-io/go-choria/choria"
	provclient "github.com/choria-io/go-choria/client/choria_provisionclient"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/prometheus/client_golang/prometheus"
)

func (h *Host) rpcUtilClient(ctx context.Context, action string, tries int, cb func(context.Context, *rpcutilclient.RpcutilClient) error) error {
	client, err := rpcutilclient.New(h.fw, rpcutilclient.Logger(h.log))
	if err != nil {
		return err
	}

	client.OptionWorkers(1).OptionTargets([]string{h.Identity})

	err = h.rpcWrapper(ctx, fmt.Sprintf("rpcutil#%s", action), tries, func(ctx context.Context) error {
		return cb(ctx, client)
	})
	if err != nil {
		return fmt.Errorf("rpc_util#%s failed: %s", action, err)
	}

	return nil
}

func (h *Host) provisionClient(ctx context.Context, action string, tries int, cb func(context.Context, *provclient.ChoriaProvisionClient) error) error {
	client, err := provclient.New(h.fw, provclient.Logger(h.log))
	if err != nil {
		return err
	}

	client.OptionWorkers(1).OptionTargets([]string{h.Identity})

	err = h.rpcWrapper(ctx, fmt.Sprintf("choria_provision#%s", action), tries, func(ctx context.Context) error {
		return cb(ctx, client)
	})
	if err != nil {
		return fmt.Errorf("choria_provision#%s failed: %s", action, err)
	}

	return nil
}

func (h *Host) rpcWrapper(ctx context.Context, action string, tries int, cb func(context.Context) error) error {
	if h.cfg.Paused() {
		return fmt.Errorf("provisioning is paused, cannot perform %s", action)
	}

	tctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := backoff.Default.For(tctx, func(try int) error {
		h.log.Debugf("Trying action %s try %d/%d", action, try, tries)
		if try > tries {
			h.log.Errorf("Maximum %d tries reached for action %s", tries, action)
			cancel()
			return fmt.Errorf("maximum tries reached")
		}

		if h.cfg.Paused() {
			cancel()
			return fmt.Errorf("provisioning is paused, cannot perform %s", action)
		}

		obs := prometheus.NewTimer(rpcDuration.WithLabelValues(h.cfg.Site, action))
		defer obs.ObserveDuration()

		err := cb(tctx)
		if err != nil {
			h.log.Errorf("rpc handler for %s failed: %s", action, err)
			return fmt.Errorf("rpc handler failed: %s", err)
		}

		return nil
	})
	if err != nil {
		rpcErrCtr.WithLabelValues(h.cfg.Site, action).Inc()
	}

	return err
}

func (h *Host) upgrade(ctx context.Context) error {
	return h.provisionClient(ctx, "release_update", 3, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Infof("Upgrading node to %s", h.upgradeTargetVersion)

		res, err := pc.ReleaseUpdate(h.cfg.UpgradesRepo, h.token, h.upgradeTargetVersion).Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not perform upgrade: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.ReleaseUpdateOutput) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			h.log.Infof("Restart response: %s", r.Message())
		})

		return err
	})

}

func (h *Host) restart(ctx context.Context) error {
	return h.provisionClient(ctx, "restart", 3, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Restarting node")

		res, err := pc.Restart(h.token).Splay(1).Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not perform restart: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.RestartOutput) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			h.log.Infof("Restart response: %s", r.Message())
		})

		return err
	})
}

func (h *Host) shutdown(ctx context.Context) error {
	return h.provisionClient(ctx, "shutdown", 3, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Shutting down node")

		res, err := pc.Shutdown(h.token).Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not perform shutdown: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.ShutdownOutput) {
			if !r.ResultDetails().OK() {
				h.log.Warnf("Could not shutdown %v: %v (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			helperShutdownCtr.WithLabelValues(h.cfg.Site).Inc()

			h.log.Infof("Shutdown response: %s", r.Message())
		})

		return err
	})
}

func (h *Host) configure(ctx context.Context) error {
	return h.provisionClient(ctx, "configure", 5, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		if len(h.config) == 0 {
			return fmt.Errorf("empty configuration")
		}

		h.log.Info("Configuring node")

		cj, err := json.Marshal(h.config)
		if err != nil {
			return fmt.Errorf("could not encode configuration: %s", err)
		}

		req := pc.Configure(string(cj)).
			Token(h.token).
			Ca(h.ca).
			Certificate(h.cert).
			Ssldir(h.sslDir).
			Key(h.key).
			EcdhPublic(h.provisionPubKey).
			ActionPolicies(h.actionPolicies).
			OpaPolicies(h.opaPolicies).
			ServerJwt(h.signedServerJWT)

		if h.CSR != nil && h.CSR.SSLDir != "" {
			req.Ssldir(h.CSR.SSLDir)
		}

		res, err := req.Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not configure: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.ConfigureOutput) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			h.log.Infof("Configure response: %s", r.Message())
		})

		return err
	})
}

func (h *Host) fetchEd25519PubKey(ctx context.Context) error {
	if h.edPubK != "" && h.sslDir != "" && h.ED25519PubKey != nil {
		h.log.Infof("Already have ED25519 public key, not retrieving again")
		return nil
	}

	return h.provisionClient(ctx, "gen25519", 5, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Infof("Fetching ED25519 public key")
		h.nonce, _ = choria.NewRequestID()

		res, err := pc.Gen25519(h.nonce, h.token).Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not retrieve ED25519 public key: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.Gen25519Output) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			if len(r.PublicKey()) == 0 {
				err = fmt.Errorf("no public key received")
				return
			}
			if len(r.Signature()) == 0 {
				err = fmt.Errorf("no signature received")
				return
			}

			var (
				pk  []byte
				sig []byte
			)

			pk, err = hex.DecodeString(r.PublicKey())
			if err != nil {
				err = fmt.Errorf("invalid public key received: %s", err)
				return
			}

			sig, err = hex.DecodeString(r.Signature())
			if err != nil {
				err = fmt.Errorf("invalid signature")
				return
			}

			if !ed25519.Verify(pk, []byte(h.nonce), sig) {
				err = fmt.Errorf("invalid nonce signature")
				return
			}

			h.ED25519PubKey = &provision.ED25519Reply{}
			err = r.ParseGen25519Output(h.ED25519PubKey)
			if err != nil {
				err = fmt.Errorf("could not parse gen25519 reply: %s", err)
				return
			}

			h.edPubK = r.PublicKey()
			h.sslDir = r.Directory()
		})

		if err != nil {
			h.log.Errorf("Could not fetch ed25519 public key: %s", err)
		}

		return err
	})
}

func (h *Host) fetchJWT(ctx context.Context) (err error) {
	if h.rawJWT != "" {
		h.log.Infof("Already have JWT for %s, not retrieving again", h.Identity)
		return nil
	}

	return h.provisionClient(ctx, "jwt", 3, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Fetching JWT")

		res, err := pc.Jwt(h.token).Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not retrieve JWT: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.JwtOutput) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			h.rawJWT = r.Jwt()
			h.serverPubKey = r.EcdhPublic()
		})

		return err
	})
}

func (h *Host) fetchInventory(ctx context.Context) (err error) {
	if len(h.Metadata) > 0 {
		h.log.Infof("Already have metadata for %s, not retrieving again", h.Identity)
		return nil
	}

	return h.rpcUtilClient(ctx, "inventory", 5, func(ctx context.Context, rpcc *rpcutilclient.RpcutilClient) error {
		h.log.Info("Fetching Inventory")

		res, err := rpcc.Inventory().Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not retrieve inventory: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *rpcutilclient.InventoryOutput) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			var j []byte
			j, err = r.JSON()
			if err != nil {
				err = fmt.Errorf("could not obtain inventory JSON data: %s", err)
				return
			}

			h.Metadata = string(j)
			h.version = r.Version()
			h.upgradable = r.Upgradable()
		})

		return err
	})

}

func (h *Host) fetchCSR(ctx context.Context) error {
	return h.provisionClient(ctx, "gencsr", 1, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Fetching CSR")

		res, err := pc.Gencsr(h.token).Cn(h.Identity).Do(ctx)
		if err != nil {
			return err
		}

		if res.Stats().ResponsesCount() != 1 {
			return fmt.Errorf("could not fetch CSR: received %d responses while expecting a response from %s", res.Stats().ResponsesCount(), h.Identity)
		}

		res.EachOutput(func(r *provclient.GencsrOutput) {
			if !r.ResultDetails().OK() {
				err = fmt.Errorf("invalid response from %s: %s (%d)", r.ResultDetails().Sender(), r.ResultDetails().StatusMessage(), r.ResultDetails().StatusCode())
				return
			}

			h.CSR = &provision.CSRReply{}
			err = r.ParseGencsrOutput(h.CSR)
			if err != nil {
				h.log.Errorf("Could not parse reply from %s: %s", r.ResultDetails().Sender(), err)
			}
		})

		return err
	})
}
