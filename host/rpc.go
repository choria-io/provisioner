package host

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/choria-io/go-choria/backoff"
	provclient "github.com/choria-io/go-choria/client/choria_provisionclient"
	"github.com/choria-io/go-choria/client/rpcutilclient"
	"github.com/choria-io/go-choria/providers/agent/mcorpc/golang/provision"
	"github.com/prometheus/client_golang/prometheus"
)

func (h *Host) rpcUtilClient(ctx context.Context, action string, tries int, cb func(context.Context, *rpcutilclient.RpcutilClient) error) error {
	client, err := rpcutilclient.New(rpcutilclient.Choria(h.fw), rpcutilclient.Logger(h.log))
	if err != nil {
		return err
	}

	client.OptionWorkers(1).OptionTargets([]string{h.Identity})

	return h.rpcWrapper(ctx, fmt.Sprintf("rpcutil#%s", action), tries, func(ctx context.Context) error {
		return cb(ctx, client)
	})
}

func (h *Host) provisionClient(ctx context.Context, action string, tries int, cb func(context.Context, *provclient.ChoriaProvisionClient) error) error {
	client, err := provclient.New(provclient.Choria(h.fw), provclient.Logger(h.log))
	if err != nil {
		return err
	}

	client.OptionWorkers(1).OptionTargets([]string{h.Identity})

	return h.rpcWrapper(ctx, fmt.Sprintf("choria_provision#%s", action), tries, func(ctx context.Context) error {
		return cb(ctx, client)
	})
}

func (h *Host) rpcWrapper(ctx context.Context, action string, tries int, cb func(context.Context) error) error {
	if h.cfg.Paused() {
		return fmt.Errorf("provisioning is paused, cannot perform %s", action)
	}

	tctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := backoff.Default.For(tctx, func(try int) error {
		if try > tries {
			cancel()
			return fmt.Errorf("maximum tries reached")
		}

		if h.cfg.Paused() {
			cancel()
			return fmt.Errorf("provisioning is paused, cannot perform %s", action)
		}

		obs := prometheus.NewTimer(rpcDuration.WithLabelValues(h.cfg.Site, action))
		defer obs.ObserveDuration()

		return cb(tctx)
	})
	if err != nil {
		rpcErrCtr.WithLabelValues(h.cfg.Site, action).Inc()
	}

	return err
}

func (h *Host) restart(ctx context.Context) error {
	return h.provisionClient(ctx, "restart", 1, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Restarting node")

		res, err := pc.Restart().Token(h.token).Splay(1).Do(ctx)
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

func (h *Host) configure(ctx context.Context) error {
	return h.provisionClient(ctx, "configure", 1, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		if len(h.config) == 0 {
			return fmt.Errorf("empty configuration")
		}

		h.log.Info("Configuring node")

		cj, err := json.Marshal(h.config)
		if err != nil {
			return fmt.Errorf("could not encode configuration: %s", err)
		}

		req := pc.Configure(string(cj)).Token(h.token).Ca(h.ca).Certificate(h.cert)
		if h.CSR != nil {
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

func (h *Host) fetchJWT(ctx context.Context) (err error) {
	if h.rawJWT != "" {
		h.log.Infof("Already have JWT for %s, not retrieving again", h.Identity)
		return nil
	}

	return h.provisionClient(ctx, "jwt", 5, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Fetching JWT")

		res, err := pc.Jwt().Token(h.token).Do(ctx)
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
		})

		return err
	})
}

func (h *Host) fetchInventory(ctx context.Context) (err error) {
	if len(h.Metadata) > 0 {
		h.log.Infof("Already have metadata for %s, not retrieving again", h.Identity)
		return nil
	}

	return h.rpcUtilClient(ctx, "jwt", 5, func(ctx context.Context, rpcc *rpcutilclient.RpcutilClient) error {
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

			j, err := r.JSON()
			if err != nil {
				err = fmt.Errorf("could not obtain inventory JSON data: %s", err)
				return
			}

			h.Metadata = string(j)
		})

		return err
	})

}

func (h *Host) fetchCSR(ctx context.Context) error {
	return h.provisionClient(ctx, "choria_provision#gencsr", 1, func(ctx context.Context, pc *provclient.ChoriaProvisionClient) error {
		h.log.Info("Fetching CSR")

		res, err := pc.Gencsr().Token(h.token).Cn(h.Identity).Do(ctx)
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
