+++
title = "The provisioning cycle"
weight = 30
description = "What Host.Provision does to a server, which steps the feature flags gate, and what the delay flag controls."
+++

`Host.Provision` runs on a worker goroutine, holds the host's mutex for the duration of the call, and
returns a delay flag and an error.

{{% notice style="note" title="Where it lives" %}}
`host`: everything that happens to one server. Main file: `host/host.go`, with `Provision` at
`host/host.go:78` and `handleHostUpgrade` at `host/host.go:219`.
{{% /notice %}}

## The sequence

The feature flags gate individual steps. The order never changes.

<figure class="cm-diagram">
  <svg viewBox="0 0 760 350" role="img" aria-label="The provisioning sequence. The first row fetches the JWT, the ed25519 key, the inventory, and the CSR. The second row calls the helper, optionally upgrades, then configures and restarts. The cycle ends in one of three ways: defer or shutdown from the helper, an upgrade that requeues the node, or a successful provision.">
    <defs>
      <marker id="pcah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="pcah3" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent3)"/></marker>
      <marker id="pcah2" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent2)"/></marker>
    </defs>
    <!-- row one: gather -->
    <rect class="cm-svg-box" x="30" y="50" width="160" height="58" rx="8"/>
    <text class="cm-svg-label" x="110" y="76" text-anchor="middle">Fetch JWT</text>
    <text class="cm-svg-sub" x="110" y="94" text-anchor="middle">features.jwt</text>
    <rect class="cm-svg-box" x="210" y="50" width="160" height="58" rx="8"/>
    <text class="cm-svg-label" x="290" y="76" text-anchor="middle">Fetch ed25519</text>
    <text class="cm-svg-sub" x="290" y="94" text-anchor="middle">features.ed25519</text>
    <rect x="390" y="50" width="160" height="58" rx="8" fill="color-mix(in srgb, var(--cm-accent) 12%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="470" y="76" text-anchor="middle" style="fill:var(--cm-accent)">Fetch inventory</text>
    <text class="cm-svg-sub" x="470" y="94" text-anchor="middle">always</text>
    <rect class="cm-svg-box" x="570" y="50" width="160" height="58" rx="8"/>
    <text class="cm-svg-label" x="650" y="76" text-anchor="middle">Fetch CSR</text>
    <text class="cm-svg-sub" x="650" y="94" text-anchor="middle">features.pki</text>
    <line x1="190" y1="79" x2="206" y2="79" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <line x1="370" y1="79" x2="386" y2="79" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <line x1="550" y1="79" x2="566" y2="79" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <!-- wrap to row two -->
    <path d="M650,108 L650,148 L110,148 L110,186" fill="none" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <!-- row two: decide and act -->
    <rect x="30" y="190" width="160" height="58" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="110" y="216" text-anchor="middle" style="fill:var(--cm-accent)">Call helper</text>
    <text class="cm-svg-sub" x="110" y="234" text-anchor="middle">always</text>
    <rect class="cm-svg-box" x="210" y="190" width="160" height="58" rx="8"/>
    <text class="cm-svg-label" x="290" y="216" text-anchor="middle">Upgrade</text>
    <text class="cm-svg-sub" x="290" y="234" text-anchor="middle">features.upgrades</text>
    <rect x="390" y="190" width="160" height="58" rx="8" fill="color-mix(in srgb, var(--cm-accent) 12%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="470" y="216" text-anchor="middle" style="fill:var(--cm-accent)">Configure</text>
    <text class="cm-svg-sub" x="470" y="234" text-anchor="middle">always</text>
    <rect x="570" y="190" width="160" height="58" rx="8" fill="color-mix(in srgb, var(--cm-accent) 12%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="650" y="216" text-anchor="middle" style="fill:var(--cm-accent)">Restart</text>
    <text class="cm-svg-sub" x="650" y="234" text-anchor="middle">always</text>
    <line x1="190" y1="219" x2="206" y2="219" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <line x1="370" y1="219" x2="386" y2="219" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <line x1="550" y1="219" x2="566" y2="219" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
    <!-- outcomes -->
    <rect x="30" y="292" width="160" height="48" rx="8" fill="color-mix(in srgb, var(--cm-accent3) 14%, transparent)" stroke="var(--cm-accent3)"/>
    <text class="cm-svg-label" x="110" y="314" text-anchor="middle" style="fill:var(--cm-accent3)">Defer or shutdown</text>
    <text class="cm-svg-sub" x="110" y="331" text-anchor="middle">helper decided</text>
    <rect x="210" y="292" width="160" height="48" rx="8" fill="color-mix(in srgb, var(--cm-accent2) 14%, transparent)" stroke="var(--cm-accent2)"/>
    <text class="cm-svg-label" x="290" y="314" text-anchor="middle" style="fill:var(--cm-accent2)">Requeue</text>
    <text class="cm-svg-sub" x="290" y="331" text-anchor="middle">upgraded, no delay</text>
    <rect x="570" y="292" width="160" height="48" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="650" y="314" text-anchor="middle" style="fill:var(--cm-accent)">Provisioned</text>
    <text class="cm-svg-sub" x="650" y="331" text-anchor="middle">60 second hold</text>
    <line x1="110" y1="248" x2="110" y2="288" stroke="var(--cm-accent3)" stroke-width="2" marker-end="url(#pcah3)"/>
    <line x1="290" y1="248" x2="290" y2="288" stroke="var(--cm-accent2)" stroke-width="2" marker-end="url(#pcah2)"/>
    <line x1="650" y1="248" x2="650" y2="288" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#pcah)"/>
  </svg>
  <figcaption>Defer, shutdown, and a completed upgrade all end the cycle before <code>configure</code> runs.</figcaption>
</figure>

## Checks before the first RPC

`Provision` checks `provisioned` and discovery age before it issues any RPC.

```go
if h.provisioned {
	return true, nil
}

if !h.discovered.IsZero() {
	since := time.Since(h.discovered)
	if since > 2*h.cfg.IntervalDuration {
		return false, fmt.Errorf("skipping node that's been waiting %v", since)
	}
}
```

The `provisioned` short-circuit returns a delay, so the finisher holds a node that has already been
through the sequence out of the map for a further 60 seconds.

The staleness check is the second half of the duplicate-provision trade-off described in
[Discovery and the work queue]({{% relref "discovery" %}}). Because the queue holds up to 50,000
entries, a host can wait long enough that the facts the helper would see no longer match the node.
`Provision` refuses a host that has waited longer than two
discovery intervals.

## Gathering facts

Each fetch is a single RPC against one target, described in
[RPC, retries, and upgrades]({{% relref "rpc-and-upgrades" %}}). `fetchJWT`, `fetchInventory`, and
`fetchEd25519PubKey` each return early when the value they collect is already populated, so a retried
`Provision` on the same `Host` does not re-fetch.

`fetchInventory` always runs. Besides the facts, it records `h.version` and `h.upgradable` from the
reply, and the upgrade path requires both.

## The helper's answer

`getConfig` marshals the whole `Host` to JSON and runs the helper. Its reply, a
`host.ConfigResponse`, can end the cycle:

<dl class="cm-kv">
  <dt>defer</dt><dd>Returns an error carrying the helper's message. The node stays unconfigured. The finisher removes it from the map, and a later discovery cycle picks it up again.</dd>
  <dt>shutdown</dt><dd>Issues a <code>shutdown</code> RPC to the node and returns a delay. The node exits with code 0, which systemd does not restart. When <code>msg</code> is empty, <code>Provision</code> logs a warning that carries no shutdown reason.</dd>
</dl>

`Provision` copies the rest of the reply onto the `Host`: the configuration map, the CA, certificate,
key, and SSL directory, the target upgrade version, and the two policy maps.
[The helper contract]({{% relref "helper" %}}) gives the full shape.

## The upgrade interruption

When `features.upgrades` is on and the helper returned an `upgrade` version, `handleHostUpgrade` runs
before configuration. It refuses to proceed if the inventory carried no version or the node reported
`upgradable` as false, then compares versions with the RPM comparison in `host/version.go`.

The switch that consumes its result (`host/host.go:189-202`) handles these outcomes:

| Result | Effect |
|--------|--------|
| No error, `skipped` true | Versions already match. Falls through and configures the node normally. |
| No error, `skipped` false | An upgrade ran. Returns `(false, nil)` immediately, ending the cycle before configuration. |
| Error, `upgrades_optional` true | Logs a warning and configures the node on its old version. |
| Error, `upgrades_optional` false | Returns `(true, err)`, so the finisher holds the node out of the map for 60 seconds. |

{{% notice style="warning" title="Upgrade and configure never share a pass" %}}
An upgraded node returns a false delay flag so the next discovery cycle picks it up at once ("no delay
so we reprov asap"). The node has already replaced its own binary and restarted into provisioning mode
on the new version. Returning no delay leaves it eligible for the next discovery cycle, where it runs
the full sequence again and this time reaches `configure`. A node is never both upgraded and
configured in a single pass.
{{% /notice %}}

## Configure and restart

`configure` sends the whole payload in one RPC: the configuration map as a JSON string, the token, the
CA, certificate, key, SSL directory, the Provisioner's ECDH public key, both policy maps, and the
signed server JWT. It refuses to send an empty configuration map. If the CSR reply carried an SSL
directory, that value overrides whatever the helper chose.

`restart` follows with a one second splay, and the node restarts into its new configuration.
`Provision` sets `h.provisioned` to true after the restart returns, then returns `(true, nil)`.

{{% notice style="tip" title="Next" %}}
[The helper contract]({{% relref "helper" %}}) covers the JSON exchange.
[RPC, retries, and upgrades]({{% relref "rpc-and-upgrades" %}}) covers what happens inside each call.
{{% /notice %}}
