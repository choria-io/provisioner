+++
title = "The helper contract"
weight = 40
description = "The JSON exchange between Provisioner and site-specific logic, and how the child process is run."
+++

Provisioner ships no knowledge of any particular site. It cannot name a node, reach a certificate
authority, pick a broker, or decide which policies a machine should carry, because every one of those
answers differs between deployments and most of them live in systems Provisioner has never heard of.
The helper supplies them. It is a program in any language, named by the `helper` configuration key,
that reads a JSON description of one server on standard input and writes that server's configuration
back on standard output.

The helper runs at the midpoint of the provisioning cycle, once per server. By the time Provisioner
calls it, the fetch steps have already collected everything the node can say about itself: its
verified `provisioning.jwt`, its inventory, and its CSR or ed25519 public key, depending on the
enrollment model. The helper turns that description into a decision. It can return a configuration
map, x509 credentials, a signed-token request, Open Policy Agent and Action Policy documents, and a
version to upgrade to. It can also refuse: `defer` leaves the node unconfigured for a later cycle, and
`shutdown` stops it. Provisioner takes whatever comes back, sends it to the node in a single
`configure` call, and restarts the node.

{{% notice style="note" title="Where it lives" %}}
`host/helper.go`, 131 lines. `getConfig` at `host/helper.go:38` marshals the input,
`runDecodedHelper` decodes the reply, and `runHelper` at `host/helper.go:68` starts the child process
and waits on it.
{{% /notice %}}

## One exec per server

<figure class="cm-diagram">
  <svg viewBox="0 0 760 300" role="img" aria-label="The Host struct is marshaled to JSON and written to the helper process on standard input. The helper writes JSON back on standard output, which is decoded into a ConfigResponse. Provisioner checks the pause flag before the process starts, the whole call runs under a 10 second context timeout, and empty output or a non-zero exit is a failure.">
    <defs>
      <marker id="hpah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="hpah3" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent3)"/></marker>
    </defs>
    <!-- pause gate -->
    <rect x="290" y="15" width="180" height="48" rx="8" fill="color-mix(in srgb, var(--cm-accent2) 14%, transparent)" stroke="var(--cm-accent2)"/>
    <text class="cm-svg-label" x="380" y="37" text-anchor="middle" style="fill:var(--cm-accent2)">Pause gate</text>
    <text class="cm-svg-sub" x="380" y="54" text-anchor="middle">no exec while paused</text>
    <line x1="380" y1="63" x2="380" y2="91" stroke="var(--cm-accent2)" stroke-width="2" marker-end="url(#hpah)"/>
    <!-- timeout bracket -->
    <rect x="272" y="78" width="216" height="116" rx="10" fill="none" stroke="var(--cm-faint)" stroke-width="1.5" stroke-dasharray="6 4"/>
    <!-- input -->
    <rect class="cm-svg-box" x="20" y="100" width="160" height="76" rx="8"/>
    <text class="cm-svg-label" x="100" y="126" text-anchor="middle">Host struct</text>
    <text class="cm-svg-sub" x="100" y="145" text-anchor="middle">identity, csr, jwt,</text>
    <text class="cm-svg-sub" x="100" y="161" text-anchor="middle">ed25519, inventory</text>
    <!-- helper -->
    <rect x="290" y="95" width="180" height="82" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="380" y="122" text-anchor="middle" style="fill:var(--cm-accent)">helper process</text>
    <text class="cm-svg-sub" x="380" y="141" text-anchor="middle">any language</text>
    <text class="cm-svg-sub" x="380" y="157" text-anchor="middle">argv via shellquote</text>
    <!-- output -->
    <rect class="cm-svg-box" x="580" y="100" width="160" height="76" rx="8"/>
    <text class="cm-svg-label" x="660" y="126" text-anchor="middle">ConfigResponse</text>
    <text class="cm-svg-sub" x="660" y="145" text-anchor="middle">12 fields, all</text>
    <text class="cm-svg-sub" x="660" y="161" text-anchor="middle">optional</text>
    <!-- edges -->
    <line x1="180" y1="136" x2="286" y2="136" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#hpah)"/>
    <text class="cm-svg-sub" x="233" y="128" text-anchor="middle">stdin</text>
    <line x1="470" y1="136" x2="576" y2="136" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#hpah)"/>
    <text class="cm-svg-sub" x="523" y="128" text-anchor="middle">stdout</text>
    <text class="cm-svg-sub" x="380" y="212" text-anchor="middle">context timeout: 10 seconds, not configurable</text>
    <!-- failure -->
    <rect x="290" y="240" width="180" height="48" rx="8" fill="color-mix(in srgb, var(--cm-accent3) 14%, transparent)" stroke="var(--cm-accent3)"/>
    <text class="cm-svg-label" x="380" y="262" text-anchor="middle" style="fill:var(--cm-accent3)">Failure</text>
    <text class="cm-svg-sub" x="380" y="279" text-anchor="middle">empty output or exit != 0</text>
    <line x1="380" y1="222" x2="380" y2="236" stroke="var(--cm-accent3)" stroke-width="2" marker-end="url(#hpah3)"/>
    <text class="cm-svg-sub" x="620" y="250" text-anchor="middle">helper stderr is discarded</text>
    <text class="cm-svg-sub" x="620" y="266" text-anchor="middle">by os/exec, not logged</text>
  </svg>
  <figcaption>A fresh process per server, so no state carries over between calls.</figcaption>
</figure>

## What the helper receives

`getConfig` marshals the `Host` itself, so the input shape is exactly the struct's exported fields
(`host/host.go:29-51`). The JSON-tagged fields are the only ones the helper sees; every other field on
`Host` is unexported and stays in the process.

<dl class="cm-kv">
  <dt>identity</dt><dd>The Choria Server identity, typically the fully qualified domain name.</dd>
  <dt>csr</dt><dd>A <code>provision.CSRReply</code>, non-null only when <code>features.pki</code> is enabled.</dd>
  <dt>ed25519_pubkey</dt><dd>A <code>provision.ED25519Reply</code>, non-null only when <code>features.ed25519</code> is enabled.</dd>
  <dt>inventory</dt><dd>The <code>rpcutil#inventory</code> result. A JSON string, not a nested object.</dd>
  <dt>jwt</dt><dd>The parsed and already-verified <code>provisioning.jwt</code> claims, non-null only when <code>features.jwt</code> is enabled.</dd>
</dl>

{{% notice style="warning" title="Caveat" %}}
`inventory` is a string containing JSON, because `fetchInventory` assigns
`h.Metadata = string(j)` from the RPC reply. A helper has to decode it a second time to reach the
facts, the agent list, or the version. The `identity` value at the top level and the one inside the
inventory come from different sources, so a helper that needs the identity should use the top-level
field.
{{% /notice %}}

The token, the signing keys, and the helper path itself are not sent. A helper that needs a secret
reads it from its own environment or files.

## What the helper returns

The reply decodes into `host.ConfigResponse` (`host/helper.go:23`). Unknown keys are ignored and every
field is optional, so a minimal helper answers with a `configuration` object alone.

```go
type ConfigResponse struct {
	Defer          bool                 `json:"defer"`
	Shutdown       bool                 `json:"shutdown"`
	Msg            string               `json:"msg"`
	Key            string               `json:"key"`
	Certificate    string               `json:"certificate"`
	CA             string               `json:"ca"`
	SSLDir         string               `json:"ssldir"`
	ServerClaims   *tokens.ServerClaims `json:"server_claims"`
	Configuration  map[string]string    `json:"configuration"`
	ActionPolicies map[string]string    `json:"action_policies"`
	OPAPolicies    map[string]string    `json:"opa_policies"`
	UpgradeVersion string               `json:"upgrade"`
}
```

`Provision` checks `defer` and `shutdown` first, and either one ends the cycle. `upgrade` diverts to
the upgrade path. `Provision` copies the rest onto the `Host` and sends it to the node in the single
`configure` RPC. [The provisioning cycle]({{% relref "provisioning-cycle" %}}) gives the order those
checks run in, and [Writing a helper]({{% relref "/configuration/helper" %}}) gives worked examples in
Ruby.

## The child process

`runHelper` runs these steps in order.

<ol class="cm-steps">
  <li><b>Timing starts first</b> A <code>prometheus.NewTimer</code> on <code>choria_provisioner_helper_time</code> is deferred before anything else, so a timed-out helper still records its duration.</li>
  <li><b>The pause flag is checked</b> A paused instance refuses to start the helper, and the error includes the helper path.</li>
  <li><b>A 10 second timeout is applied</b> <code>context.WithTimeout(ctx, 10*time.Second)</code> wraps the whole call. No configuration key sets the value.</li>
  <li><b>The command string is split with shellquote</b> <code>shellquote.Split</code> handles quoting, so <code>helper</code> can carry arguments, for example <code>/opt/prov/helper.rb --site london</code>. An empty result is rejected.</li>
  <li><b>Input is written from a goroutine</b> The write to stdin runs concurrently with the read from stdout and closes the pipe on completion. Doing both in one goroutine would deadlock on any helper whose output exceeds the pipe buffer.</li>
  <li><b>Output is read to completion, then the process is waited on</b> Reading before <code>Wait</code> is required: <code>Wait</code> closes the pipes.</li>
  <li><b>Zero bytes is an error</b> A helper that exits cleanly without writing anything fails with "zero bytes received" rather than producing an empty configuration.</li>
  <li><b>The exit code is checked</b> <code>ProcessState.Success()</code> must be true.</li>
</ol>

{{% notice style="warning" title="Caveat" %}}
`runHelper` sets a stdin and a stdout pipe but leaves `Stderr` nil, and `os/exec` connects a nil
`Stderr` to the null device. Anything a helper writes to standard error is discarded and never reaches
the Provisioner log. A helper that has a reason to report should return `defer` with a `msg`, which is
logged, or write its own log file.
{{% /notice %}}

`Provision` counts the failures. It increments `choria_provisioner_helper_errors` when `getConfig`
returns an error (`host/host.go:134`), and the successful shutdown path increments
`choria_provisioner_helper_shutdown_requests` from `host/rpc.go:169`.

{{% notice style="tip" title="Next" %}}
[Security and enrollment]({{% relref "security" %}}) covers what happens to the key material that the
helper returns.
{{% /notice %}}
