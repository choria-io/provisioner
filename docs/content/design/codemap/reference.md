+++
title = "Reference and map"
weight = 80
description = "The command line flags, every source file and what it holds, the exported metrics, and a glossary."
+++

Each table links to the page that explains the area it covers.

{{% notice style="note" title="Where it lives" %}}
The whole repository, from `main.go` down. `ABTaskFile`, `Rakefile`, and `packager/` hold the build
tooling.
{{% /notice %}}

## Command line

`main.go` calls `cmd.Run`, which builds a `fisk` application named `choria-provisioner`. The `run`
command is the default, so `choria-provisioner --config ...` and
`choria-provisioner run --config ...` are equivalent.

| Flag | Command | Required | Default | Effect |
|------|---------|----------|---------|--------|
| `--debug` | Global | No | Off | Forces the Choria log level to `debug`. |
| `--config` | `run` | Yes | None | Names the Provisioner YAML. Must be an existing file. |
| `--choria-config` | `run` | No | `choria.UserConfig()` | Names the Choria client configuration. Must be an existing file. |
| `--pid` | `run` | No | Unset | Writes the PID, and removes the file on exit. |

`--version` prints `config.Version`, which is `0.0.0` in a plain `go build`. The release build injects
it through `-ldflags`, mapped in `packager/buildspec.yaml` to
`github.com/choria-io/provisioner/config.Version`.

## Source map

| File | Lines | Holds |
|------|-------|-------|
| `main.go` | 9 | The call to `cmd.Run`. |
| `cmd/provisioner.go` | 130 | Flags, both config loads, the Choria framework, `/metrics`, signal handling, and the PID file. [Architecture]({{% relref "architecture" %}}). |
| `config/config.go` | 142 | The `Config` struct, `Load`, the defaults, and the validation. |
| `config/pausable.go` | 48 | `Pause`, `Resume`, `Flip`, and `Paused`. [Pausing and leader election]({{% relref "clustering" %}}). |
| `config/stats.go` | 18 | The `choria_provisioner_paused` gauge. |
| `hosts/hosts.go` | 223 | `Process`, the package state, broadcast discovery, `add`, `remove`, and `isCurrent`. [Discovery and the work queue]({{% relref "discovery" %}}). |
| `hosts/event.go` | 82 | The connector and the lifecycle startup subscription. |
| `hosts/provisioner.go` | 84 | The worker loop and the finisher. |
| `hosts/election.go` | 51 | The leader election setup and its two callbacks. |
| `hosts/stats.go` | 66 | Nine fleet-level counters and gauges. |
| `host/host.go` | 483 | The `Host` struct, `Provision`, JWT and CSR validation, server JWT issuance, and key encryption. [The provisioning cycle]({{% relref "provisioning-cycle" %}}) and [Security and enrollment]({{% relref "security" %}}). |
| `host/rpc.go` | 404 | `rpcWrapper` and every Choria RPC action. [RPC, retries, and upgrades]({{% relref "rpc-and-upgrades" %}}). |
| `host/helper.go` | 131 | `ConfigResponse` and the child process. [The helper contract]({{% relref "helper" %}}). |
| `host/version.go` | 210 | The vendored RPM version comparison. |
| `host/stats.go` | 42 | Five per-server metrics. |
| `host/host_test.go` | 238 | Ginkgo specs for `generateServerJWT`, `validateCSR`, and `encryptPrivateKey`. |
| `tools.go` | 18 | Build-tagged `tools` imports that pin the Ginkgo CLI, excluded from normal builds. |

## Key types

<dl class="cm-kv">
  <dt>config.Config</dt><dd><code>config/config.go:25</code>. The parsed YAML plus derived durations, the config file path, and the pause flag. Passed by pointer to every package.</dd>
  <dt>host.Host</dt><dd><code>host/host.go:29</code>. One server. Five exported fields form the helper's input. The remaining fields hold state that the RPC sequence fills in.</dd>
  <dt>host.ConfigResponse</dt><dd><code>host/helper.go:23</code>. The helper's reply. Twelve optional fields.</dd>
  <dt>host.Version</dt><dd><code>host/version.go:21</code>. An RPM epoch, version, and release triple with a <code>Compare</code> method.</dd>
</dl>

## Exported metrics

Every metric carries a `site` label taken from the `site` configuration key. Three `init` functions
register them, one per package.

| Metric | Type | Extra labels | Registered in |
|--------|------|--------------|---------------|
| `choria_provisioner_paused` | Gauge | None | `config/stats.go` |
| `choria_provisioner_discovered` | Counter | None | `hosts/stats.go` |
| `choria_provisioner_event_discovered` | Counter | None | `hosts/stats.go` |
| `choria_provisioner_discover_cycles` | Counter | None | `hosts/stats.go` |
| `choria_provisioner_discovery_errors` | Counter | None | `hosts/stats.go` |
| `choria_provisioner_provision_errors` | Counter | None | `hosts/stats.go` |
| `choria_provisioner_provisioned` | Counter | None | `hosts/stats.go` |
| `choria_provisioner_busy_workers` | Gauge | None | `hosts/stats.go` |
| `choria_provisioner_work_queue_entries` | Gauge | None | `hosts/stats.go` |
| `choria_provisioner_waiting_nodes` | Gauge | None | `hosts/stats.go` |
| `choria_provisioner_rpc_time` | Summary | `rpc` | `host/stats.go` |
| `choria_provisioner_rpc_errors` | Counter | `rpc` | `host/stats.go` |
| `choria_provisioner_helper_time` | Summary | None | `host/stats.go` |
| `choria_provisioner_helper_errors` | Counter | None | `host/stats.go` |
| `choria_provisioner_helper_shutdown_requests` | Counter | None | `host/stats.go` |

The `rpc` label is the `agent#action` string, for example `choria_provision#configure` or
`rpcutil#inventory`. [Monitoring]({{% relref "/monitoring" %}}) describes what each metric measures.

## Configuration keys

[Configuration File]({{% relref "/configuration/provisioner" %}}) gives the full operational
description. The following table gives the struct field and the reader for each key.

| YAML key | Field | Read by |
|----------|-------|---------|
| `workers` | `Workers` | `hosts.Process`, to size the worker pool. |
| `interval` | `Interval`, `IntervalDuration` | `hosts.Process` for the ticker, `add` for the dedup window, `Provision` for the staleness check. |
| `helper` | `Helper` | `runHelper`, split with `shellquote`. |
| `token` | `Token` | `NewHost`, which copies it to `Host.token` for every `choria_provision` action. |
| `site` | `Site` | Every metric, as the `site` label. |
| `lifecycle_component` | `LifecycleComponent` | `listen`, as the startup event subject. |
| `logfile`, `loglevel` | `Logfile`, `Loglevel` | `cmd.run`, which copies them onto the Choria configuration. |
| `choria_insecure` | `Insecure` | `cmd.run`, which disables TLS and forces the `file` security provider. |
| `monitor_port` | `MonitorPort` | `cmd.setupPrometheus`. |
| `broker_provisioning_password` | `BrokerProvisionPassword` | `cmd.run`, which sets the NATS user to `provisioner`. |
| `cert_deny_list` | `CertDenyList` | `validateCSR`, through `matchAnyRegex`. |
| `jwt_verify_cert` | `JWTVerifyCert` | `validateJWT`, as a file path or a hex public key. |
| `jwt_signing_key` | `JWTSigningKey` | `generateServerJWT`. |
| `jwt_signing_token` | `JWTSigningToken` | `generateServerJWT`, re-read for every server. |
| `server_jwt_validity` | `ServerJWTValidity`, `ServerJWTValidityDuration` | `generateServerJWT`. |
| `upgrades_repository` | `UpgradesRepo` | `upgrade`, as the `release_update` repository. |
| `upgrades_optional` | `UpgradesOptional` | `Provision`, in the upgrade failure branch. |
| `leader_election` | `LeaderElection` | `hosts.Process`, to start `startElection`. |
| `features.jwt` | `Features.JWT` | `Provision`, to gate `fetchJWT` and `validateJWT`. |
| `features.pki` | `Features.PKI` | `Provision`, to gate `fetchCSR` and `validateCSR`. |
| `features.ed25519` | `Features.ED25519` | `Provision`, to gate `fetchEd25519PubKey` and `generateServerJWT`. |
| `features.upgrades` | `Features.VersionUpgrades` | `Provision`, to gate `handleHostUpgrade`. |

`rego_policy` is declared on `Config` as `RegoPolicy` and read nowhere. OPA policies reach a node
through the helper's `opa_policies` reply instead.

## Choria RPC actions used

| Agent | Action | Purpose |
|-------|--------|---------|
| `rpcutil` | `inventory` | Facts, agent list, version, and the `upgradable` flag. |
| `choria_provision` | `jwt` | The `provisioning.jwt` and the server's ECDH public key. |
| `choria_provision` | `gen25519` | An ed25519 public key and a signature over a nonce. |
| `choria_provision` | `gencsr` | A CSR generated by the node. |
| `choria_provision` | `release_update` | An in-place binary upgrade. |
| `choria_provision` | `configure` | Configuration, credentials, and policies in one call. |
| `choria_provision` | `restart` | Restart into the new configuration. |
| `choria_provision` | `shutdown` | Exit with code 0, which does not trigger a systemd restart. |

## Glossary

<dl class="cm-kv">
  <dt>fleet node</dt><dd>A machine running Choria Server. Provisioner's unit of work.</dd>
  <dt>provisioning collective</dt><dd>The sub-collective that an unconfigured server joins. Provisioner forces both <code>collectives</code> and <code>main_collective</code> to this value.</dd>
  <dt>provisioning.jwt</dt><dd>A token placed on a node that enables provisioning mode, carries the broker address, and holds the shared token.</dd>
  <dt>helper</dt><dd>The site-supplied program that turns a server's identity and inventory into a configuration.</dd>
  <dt>Organization Issuer</dt><dd>The ed25519 key at the root of a certificate-authority-free Choria deployment. Provisioner signs server tokens that chain to it.</dd>
  <dt>server JWT</dt><dd>The token that Provisioner issues for a node in an Organization Issuer deployment, replacing an x509 certificate.</dd>
  <dt>lifecycle event</dt><dd>A JSON event that Choria components publish on <code>choria.lifecycle.event.&gt;</code>. Provisioner subscribes to the <code>startup</code> ones.</dd>
  <dt>leader election</dt><dd>A Choria Streams primitive that names one instance in a cluster as active.</dd>
  <dt>paused</dt><dd>The flag that a standby instance sets. Every outbound action checks it.</dd>
  <dt>splay</dt><dd>A random delay that a server applies before acting on a restart or shutdown, spreading the load across a fleet.</dd>
  <dt>site</dt><dd>A name for one installation, used as a label on every metric so a dashboard can aggregate across installations.</dd>
</dl>

{{% notice style="tip" title="Next" %}}
The [Code Map]({{% relref "/design/codemap" %}}) overview returns to the start.
[Writing a helper]({{% relref "/configuration/helper" %}}) gives the operator's view of the helper
contract.
{{% /notice %}}
