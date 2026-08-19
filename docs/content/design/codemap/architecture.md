+++
title = "Architecture"
weight = 10
description = "Package layering, the import direction, and the boundaries that put site-specific logic in the helper program."
+++

The source is four packages and about 2,400 lines.

{{% notice style="note" title="Where it lives" %}}
`main.go` calls into `cmd`. Packages: `cmd`, `config`, `hosts`, `host`.
{{% /notice %}}

## The packages

<dl class="cm-kv">
  <dt>cmd</dt><dd>The <code>fisk</code> command line, flag parsing, and process startup. Loads both configuration files, builds the Choria framework, starts the Prometheus listener and the signal handler, then calls <code>hosts.Process</code>. One file, <code>cmd/provisioner.go</code>.</dd>
  <dt>config</dt><dd>The YAML configuration, its defaults, and its validation. Also holds the pause flag that <code>rpcWrapper</code> and <code>runHelper</code> check before every outbound action. Files: <code>config.go</code>, <code>pausable.go</code>, <code>stats.go</code>.</dd>
  <dt>hosts</dt><dd>Fleet-level scheduling. Broadcast discovery, the lifecycle listener, a work queue, the worker pool, the finisher, and leader election. Files: <code>hosts.go</code>, <code>event.go</code>, <code>provisioner.go</code>, <code>election.go</code>, <code>stats.go</code>.</dd>
  <dt>host</dt><dd>Everything that happens to one server. The provisioning sequence, the RPC wrappers, the helper invocation, and RPM version comparison. Files: <code>host.go</code>, <code>rpc.go</code>, <code>helper.go</code>, <code>version.go</code>, <code>stats.go</code>.</dd>
</dl>

`cmd` imports `hosts`, `hosts` imports `host`, and all three import `config`, which imports none of
them. `host` does not import `hosts`, so `host/host_test.go` builds a `Host` from a `config.Config`
and a logger with no scheduler involved.

<figure class="cm-diagram">
  <svg viewBox="0 0 760 330" role="img" aria-label="Layered architecture. The cmd package calls hosts.Process. The hosts package contains the scheduler and imports the host package. The host package runs one server through RPC, the helper, and version comparison. All three import the config package, which imports none of them.">
    <defs>
      <marker id="arah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="arahf" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-faint)"/></marker>
    </defs>
    <!-- cmd layer -->
    <rect class="cm-svg-box" x="20" y="20" width="500" height="58" rx="8"/>
    <text class="cm-svg-label" x="270" y="44" text-anchor="middle">cmd</text>
    <text class="cm-svg-sub" x="270" y="62" text-anchor="middle">flags, choria framework, /metrics, signals</text>
    <!-- hosts layer -->
    <rect x="20" y="106" width="500" height="86" rx="8" fill="color-mix(in srgb, var(--cm-accent) 12%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="270" y="130" text-anchor="middle" style="fill:var(--cm-accent)">hosts</text>
    <text class="cm-svg-sub" x="140" y="152" text-anchor="middle">discovery + events</text>
    <text class="cm-svg-sub" x="400" y="152" text-anchor="middle">work queue + workers</text>
    <text class="cm-svg-sub" x="140" y="174" text-anchor="middle">leader election</text>
    <text class="cm-svg-sub" x="400" y="174" text-anchor="middle">finisher</text>
    <!-- host layer -->
    <rect x="20" y="220" width="500" height="86" rx="8" fill="color-mix(in srgb, var(--cm-accent) 20%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="270" y="244" text-anchor="middle" style="fill:var(--cm-accent)">host</text>
    <text class="cm-svg-sub" x="105" y="266" text-anchor="middle">Provision()</text>
    <text class="cm-svg-sub" x="270" y="266" text-anchor="middle">rpcWrapper</text>
    <text class="cm-svg-sub" x="435" y="266" text-anchor="middle">runHelper</text>
    <text class="cm-svg-sub" x="105" y="288" text-anchor="middle">JWT + keys</text>
    <text class="cm-svg-sub" x="270" y="288" text-anchor="middle">CSR checks</text>
    <text class="cm-svg-sub" x="435" y="288" text-anchor="middle">rpm versions</text>
    <!-- config sidecar -->
    <rect x="580" y="20" width="160" height="286" rx="8" fill="color-mix(in srgb, var(--cm-accent2) 14%, transparent)" stroke="var(--cm-accent2)"/>
    <text class="cm-svg-label" x="660" y="52" text-anchor="middle" style="fill:var(--cm-accent2)">config</text>
    <text class="cm-svg-sub" x="660" y="80" text-anchor="middle">YAML load</text>
    <text class="cm-svg-sub" x="660" y="100" text-anchor="middle">defaults</text>
    <text class="cm-svg-sub" x="660" y="120" text-anchor="middle">validation</text>
    <text class="cm-svg-sub" x="660" y="150" text-anchor="middle">Pause / Resume</text>
    <text class="cm-svg-sub" x="660" y="170" text-anchor="middle">Flip / Paused</text>
    <text class="cm-svg-sub" x="660" y="200" text-anchor="middle">imported by</text>
    <text class="cm-svg-sub" x="660" y="220" text-anchor="middle">all three, imports</text>
    <text class="cm-svg-sub" x="660" y="240" text-anchor="middle">none of them</text>
    <!-- import edges -->
    <line x1="270" y1="78" x2="270" y2="102" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#arah)"/>
    <text class="cm-svg-sub" x="360" y="95" text-anchor="middle">hosts.Process(ctx, cfg, fw)</text>
    <line x1="270" y1="192" x2="270" y2="216" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#arah)"/>
    <text class="cm-svg-sub" x="365" y="209" text-anchor="middle">host.NewHost / h.Provision</text>
    <line x1="576" y1="49" x2="524" y2="49" stroke="var(--cm-faint)" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#arahf)"/>
    <line x1="576" y1="149" x2="524" y2="149" stroke="var(--cm-faint)" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#arahf)"/>
    <line x1="576" y1="263" x2="524" y2="263" stroke="var(--cm-faint)" stroke-width="1.5" stroke-dasharray="4 3" marker-end="url(#arahf)"/>
  </svg>
  <figcaption><code>cmd</code> imports <code>hosts</code>, <code>hosts</code> imports <code>host</code>, and nothing imports upward, so <code>host</code> compiles and tests on its own.</figcaption>
</figure>

## The boundaries

The helper, the pause flag, and the generated RPC clients each change without touching the other two.

<ol class="cm-steps">
  <li><b>The helper</b> Provisioner marshals a <code>Host</code> to JSON, writes it to a child process on standard input, and unmarshals the reply into <code>host.ConfigResponse</code> (<code>host/helper.go:38</code>). The helper reaches certificate authorities, CMDBs, and naming services, in any language.</li>
  <li><b>The pause flag</b> <code>config.Config</code> declares <code>Pause</code>, <code>Resume</code>, <code>Flip</code>, and <code>Paused</code> (<code>config/pausable.go</code>). Leader election calls <code>Pause</code> and <code>Resume</code>, and both <code>rpcWrapper</code> and <code>runHelper</code> check the flag before sending an RPC request or starting the helper process.</li>
  <li><b>Generated RPC clients</b> <code>host/rpc.go</code> uses two clients generated in go-choria, <code>provclient.ChoriaProvisionClient</code> for the <code>choria_provision</code> agent and <code>rpcutilclient.RpcutilClient</code> for <code>rpcutil</code>. Every call to either goes through <code>rpcWrapper</code>, which holds the retries, timing, and pause check for both.</li>
</ol>

## Package-level state in `hosts`

The `hosts` package declares its state as package variables rather than struct fields
(`hosts/hosts.go:24-33`): the `hosts` map of identity to `*host.Host`, the buffered `work` and `done`
channels, a mutex, and the logger, the Choria framework, and the configuration.

```go
var (
	hosts = make(map[string]*host.Host)
	work  = make(chan *host.Host, 50000)
	done  = make(chan *host.Host, 50000)
	mu    = &sync.Mutex{}
	log   *logrus.Entry
	fw    *choria.Framework
	conf  *config.Config
	wg    = &sync.WaitGroup{}
)
```

{{% notice style="warning" title="One instance per process" %}}
`hosts.Process` writes these package variables on entry, so exactly one instance can run per process.
That matches the deployment model, where a process runs one Provisioner against one site, and the
scheduler functions take no receiver. Running two Provisioners in one binary requires moving this
state into a struct.
{{% /notice %}}

## Configuration and startup

`cmd.run` loads the Provisioner YAML from `--config` through `config.Load`, and the Choria client
configuration from `--choria-config`, defaulting to `choria.UserConfig()`. `cmd.run` then overrides
the Choria client configuration in code (`cmd/provisioner.go:61-79`): log level and log file come from
the Provisioner configuration, and the collective is forced to `provisioning` for both `Collectives`
and `MainCollective`.

Setting `choria_insecure` disables TLS, sets `protocol.Secure` to `false`, and forces the `file`
security provider. Setting `broker_provisioning_password` switches the NATS credentials to user
`provisioner` with that password. The broker's dedicated provisioning account accepts that user.

`config.Load` fills defaults and rejects configurations that cannot work
(`config/config.go:64-142`):

| Rule | Behavior |
|------|----------|
| `workers` unset | Defaults to `runtime.NumCPU()`. |
| `lifecycle_component` unset | Defaults to `provision_mode_server`. |
| `lifecycle_component` contains `.`, `>`, or `*` | Rejected, since the value is interpolated into a NATS subject. |
| `cert_deny_list` empty | Defaults to four patterns that block privileged Choria certificate names. |
| `features.pki` and `features.ed25519` both set | Rejected. The two enrollment models are exclusive. |
| `features.ed25519` set | Implies `features.jwt`. |
| `server_jwt_validity` unset | Defaults to one year. |
| `interval` below one minute | Rejected. |

## Observability

`config/stats.go`, `hosts/stats.go`, and `host/stats.go` each register their metrics in an `init`
function: the pause gauge, the fleet-level counters, and per-server timing and errors. Every metric
carries a `site` label taken from the `site` configuration key, so a query across installations can
sum or group by `site`. `cmd.setupPrometheus` serves them on `/metrics` when `monitor_port` is set.

{{% notice style="tip" title="Next" %}}
Continue to [Discovery and the work queue]({{% relref "discovery" %}}) to see how a server reaches a
worker.
{{% /notice %}}
