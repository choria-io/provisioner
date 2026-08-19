# RPC, retries, and upgrades

Every call Provisioner makes to a server goes through one wrapper, so `rpcWrapper` holds the retries,
timing, error counting, and pause check for every call.

{{% notice style="note" title="Where it lives" %}}
`host/rpc.go`, 404 lines. Every action calls `rpcWrapper` at `host/rpc.go:58`.
`host/version.go` holds the RPM version comparison that the upgrade decision uses.
{{% /notice %}}

## The shared client setup

`rpcUtilClient` and `provisionClient` (`host/rpc.go:22` and `host/rpc.go:40`) build the generated
go-choria clients for the `rpcutil` and `choria_provision` agents. Both narrow the client the same way
before calling `rpcWrapper`:

```go
client.OptionWorkers(1).OptionTargets([]string{h.Identity})
```

The client uses one worker and one target because a `Host` is one machine. The action name passed to
`rpcWrapper` is formatted as `agent#action`, and that string becomes the `rpc` label on both
`choria_provisioner_rpc_time` and `choria_provisioner_rpc_errors`, so a dashboard can show which call
is slow or failing.

## The retry loop

<figure class="cm-diagram">
  <svg viewBox="0 0 760 270" role="img" aria-label="The retry loop. A pause check runs before the loop, then backoff.For drives attempts. Each attempt starts a timer and runs the call. A failed attempt logs the error, waits for the backoff interval, and starts the next attempt; exceeding the try limit cancels the context and increments the error counter; a successful attempt returns nil.">
    <defs>
      <marker id="rlah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="rlah3" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent3)"/></marker>
      <marker id="rlah2" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent2)"/></marker>
    </defs>
    <!-- main line -->
    <rect x="20" y="40" width="160" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent2) 14%, transparent)" stroke="var(--cm-accent2)"/>
    <text class="cm-svg-label" x="100" y="64" text-anchor="middle" style="fill:var(--cm-accent2)">Pause check</text>
    <text class="cm-svg-sub" x="100" y="82" text-anchor="middle">before the loop</text>
    <rect class="cm-svg-box" x="205" y="40" width="160" height="56" rx="8"/>
    <text class="cm-svg-label" x="285" y="64" text-anchor="middle">backoff.For</text>
    <text class="cm-svg-sub" x="285" y="82" text-anchor="middle">retry driver</text>
    <rect x="390" y="40" width="160" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="470" y="64" text-anchor="middle" style="fill:var(--cm-accent)">Attempt N</text>
    <text class="cm-svg-sub" x="470" y="82" text-anchor="middle">pause, timer, call</text>
    <rect x="575" y="40" width="160" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent) 12%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="655" y="64" text-anchor="middle" style="fill:var(--cm-accent)">Success</text>
    <text class="cm-svg-sub" x="655" y="82" text-anchor="middle">returns nil</text>
    <line x1="180" y1="68" x2="201" y2="68" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#rlah)"/>
    <line x1="365" y1="68" x2="386" y2="68" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#rlah)"/>
    <line x1="550" y1="68" x2="571" y2="68" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#rlah)"/>
    <!-- failure branches -->
    <rect x="305" y="170" width="160" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent2) 14%, transparent)" stroke="var(--cm-accent2)"/>
    <text class="cm-svg-label" x="385" y="194" text-anchor="middle" style="fill:var(--cm-accent2)">Retry</text>
    <text class="cm-svg-sub" x="385" y="212" text-anchor="middle">log, then back off</text>
    <rect x="530" y="170" width="160" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent3) 14%, transparent)" stroke="var(--cm-accent3)"/>
    <text class="cm-svg-label" x="610" y="194" text-anchor="middle" style="fill:var(--cm-accent3)">Give up</text>
    <text class="cm-svg-sub" x="610" y="212" text-anchor="middle">cancel, rpc_errors + 1</text>
    <line x1="440" y1="96" x2="398" y2="166" stroke="var(--cm-accent2)" stroke-width="2" marker-end="url(#rlah2)"/>
    <line x1="505" y1="96" x2="580" y2="166" stroke="var(--cm-accent3)" stroke-width="2" marker-end="url(#rlah3)"/>
    <text class="cm-svg-sub" x="640" y="130" text-anchor="middle">try &gt; tries</text>
    <!-- loop back -->
    <path d="M305,198 L275,198 L275,100" fill="none" stroke="var(--cm-accent2)" stroke-width="2" stroke-dasharray="6 4" marker-end="url(#rlah2)"/>
    <text class="cm-svg-sub" x="380" y="252" text-anchor="middle">A pause taking effect mid-loop ends the call on the next attempt</text>
  </svg>
  <figcaption>The callback enforces the retry limit by canceling the context that runs the loop.</figcaption>
</figure>

`backoff.Default.For` retries until the callback returns nil or the context ends, so the try limit is
enforced from inside:

```go
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
```

The pause flag is read twice: once before the loop and once on every attempt. A provisioner that loses
an election mid-provision stops at the next attempt boundary rather than finishing the sequence
against a node that another instance is now provisioning.

The timer is deferred inside the callback, so `choria_provisioner_rpc_time` records each attempt
separately rather than the total across retries. `choria_provisioner_rpc_errors` increments once per
wrapper call, after the loop gives up.

## Retry limits per action

Each caller picks its own limit, and the values reflect how expensive a repeat is.

| Call | Tries | Reasoning |
|------|-------|-----------|
| `rpcutil#inventory` | 5 | Read-only and cheap to repeat. |
| `choria_provision#gen25519` | 5 | Read-only. A repeat generates a fresh nonce. |
| `choria_provision#configure` | 5 | The call that writes the configuration. Repeats are idempotent on the node. |
| `choria_provision#jwt` | 3 | Read-only. |
| `choria_provision#restart` | 3 | A repeat restarts a node that may already be restarting. |
| `choria_provision#shutdown` | 3 | A repeat targets a node that may already be shutting down. |
| `choria_provision#release_update` | 3 | A repeat downloads and rewrites the binary again. |
| `choria_provision#gencsr` | 1 | Each call makes the node generate a new private key. |

## Reply validation

Every action checks the response count first. It must be exactly one:

```go
if res.Stats().ResponsesCount() != 1 {
	return fmt.Errorf("... received %d responses while expecting a response from %s",
		res.Stats().ResponsesCount(), h.Identity)
}
```

Zero means the node did not answer within the client timeout. More than one means two machines answer
to the same identity, and the wrapper cannot tell which machine the configuration would reach, so it
fails the call. `EachOutput` then checks `ResultDetails().OK()` and assigns to a captured `err` that
the enclosing closure returns.

`shutdown` is the exception. It logs a non-OK result as a warning without setting `err`, and
increments `choria_provisioner_helper_shutdown_requests` only on the OK path. The helper already
rejected the node, so a failed shutdown confirmation does not fail the cycle.

## The upgrade decision

`handleHostUpgrade` (`host/host.go:219`) runs before configuration and refuses the node when
`upgrades_repository` is unset and `upgrades_optional` is off, when the inventory carried no version,
or when the node did not report `upgradable`.

The comparison itself uses `host/version.go`, a vendored copy of
[go-rpm-version](https://github.com/knqyf263/go-rpm-version), MIT licensed, vendored because the
upstream project is unmaintained. It implements RPM's `epoch:version-release` parsing and the
`rpmvercmp` segment comparison, including tilde handling and the rule that numeric segments sort
higher than alphabetic ones.

```go
cv := NewVersion(h.version)
tv := NewVersion(h.upgradeTargetVersion)

if !cv.Equal(tv) {
	err := h.upgrade(ctx)
	...
}
```

{{% notice style="warning" title="Version comparison is an equality test" %}}
`handleHostUpgrade` compares the two versions with `Equal` and acts on any difference. `Version`
exposes `GreaterThan` and `LessThan`, and neither is called anywhere in the codebase. A helper that
returns an `upgrade` value older than the node's current version triggers a release update that moves
the node backwards. Whether a version change is an upgrade or a rollback is therefore the helper's
decision. A helper that must prevent rollbacks compares the versions itself before setting the field.
{{% /notice %}}

`NewVersion` never returns an error. An epoch that does not parse as an integer silently becomes 0,
which follows the upstream behavior.

When the versions differ, `upgrade` sends `release_update` with the configured repository, the token,
and the target version, and `Provision` returns with no delay so the node is re-provisioned on the
next cycle.

{{% notice style="tip" title="Next" %}}
[Pausing and leader election]({{% relref "clustering" %}}) covers the pause flag that gates every
call on this page.
{{% /notice %}}
