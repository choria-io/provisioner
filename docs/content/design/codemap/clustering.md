+++
title = "Pausing and leader election"
weight = 70
description = "One boolean gates every outbound action. Only a Choria leader election sets it."
+++

Running several Provisioners against one broker would have them race to configure the same servers.
Instead, every instance starts paused and the election winner resumes.

{{% notice style="note" title="Where it lives" %}}
`config/pausable.go` holds the flag and its four methods. `hosts/election.go` calls `Pause` and
`Resume` from the election callbacks. `config/stats.go` exports the flag as a gauge.
{{% /notice %}}

## The pause flag

`config.Config` embeds a `sync.Mutex` and carries an unexported `paused` bool. `Pause`, `Resume`,
`Flip`, and `Paused` guard it, and each sets `choria_provisioner_paused` before returning:

```go
func (c *Config) Pause() {
	c.Lock()
	defer c.Unlock()

	c.paused = true
	c.setPauseStat()
}
```

`config.Load` sets the gauge to 0 at startup, so the metric exists before any election runs.

Every action that leaves the process reads the flag first:

| Reader | Effect when paused |
|--------|--------------------|
| `hosts/hosts.go:187` in `discover` | Logs and returns without sending the broadcast request. |
| `hosts/event.go:63` in `handle` | Logs and drops the lifecycle event. |
| `host/rpc.go:59` in `rpcWrapper` | Fails the call before the retry loop starts. |
| `host/rpc.go:74` inside the retry loop | Cancels the context and fails at the next attempt. |
| `host/helper.go:72` in `runHelper` | Refuses to run the helper. |

{{% notice style="warning" title="The flag is checked inside the retry loop" %}}
`rpcWrapper` reads the flag inside the retry loop as well as before it, so a Provisioner that loses an
election mid-sequence stops at the next attempt boundary. Without the inner check, an instance could
carry on sending configuration to a node that the new leader has already started working on.
{{% /notice %}}

`Flip` is defined alongside the other three and has no caller anywhere in the repository. All four
methods carry comments saying they implement `backplane.Pausable`, but this codebase does not import
the `backplane` package. The four methods satisfy the interface for a consumer that this repository
does not contain.

## Election callbacks

Setting `leader_election` makes `hosts.Process` call `conf.Pause()` before it starts any goroutine, so
the instance is inert from its first line of work. `startElection` (`hosts/election.go:17`) then
registers two callbacks with a Choria Streams election named `provisioner`.

<figure class="cm-diagram">
  <svg viewBox="0 0 760 300" role="img" aria-label="State machine with two states. An instance starts in Standby with the pause flag set. Winning the election moves it to Leader, which resumes work and sends to the discover trigger channel. Losing the election moves it back to Standby, which pauses and drains all known hosts.">
    <defs>
      <marker id="clah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="clah3" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent3)"/></marker>
    </defs>
    <!-- entry -->
    <text class="cm-svg-sub" x="160" y="32" text-anchor="middle">startup</text>
    <line x1="160" y1="40" x2="160" y2="86" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#clah)"/>
    <!-- states -->
    <rect class="cm-svg-box" x="60" y="90" width="200" height="96" rx="10" stroke-dasharray="5 4"/>
    <text class="cm-svg-label" x="160" y="122" text-anchor="middle">Standby</text>
    <text class="cm-svg-sub" x="160" y="144" text-anchor="middle">paused = 1</text>
    <text class="cm-svg-sub" x="160" y="164" text-anchor="middle">no discovery or RPC</text>
    <rect x="500" y="90" width="200" height="96" rx="10" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="600" y="122" text-anchor="middle" style="fill:var(--cm-accent)">Leader</text>
    <text class="cm-svg-sub" x="600" y="144" text-anchor="middle">paused = 0</text>
    <text class="cm-svg-sub" x="600" y="164" text-anchor="middle">provisions the fleet</text>
    <!-- transitions -->
    <path d="M260,116 C340,76 420,76 494,114" fill="none" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#clah)"/>
    <text class="cm-svg-sub" x="380" y="72" text-anchor="middle">wins the election</text>
    <path d="M500,160 C420,200 340,200 266,162" fill="none" stroke="var(--cm-accent3)" stroke-width="2" marker-end="url(#clah3)"/>
    <text class="cm-svg-sub" x="380" y="212" text-anchor="middle">loses the election</text>
    <text class="cm-svg-sub" x="160" y="206" text-anchor="middle">campaigns, backoff up to 20s</text>
    <!-- effects -->
    <rect class="cm-svg-box" x="60" y="228" width="280" height="60" rx="8"/>
    <text class="cm-svg-label" x="200" y="252" text-anchor="middle">On winning</text>
    <text class="cm-svg-sub" x="200" y="272" text-anchor="middle">Resume(), then send to discoverTrigger</text>
    <rect class="cm-svg-box" x="420" y="228" width="280" height="60" rx="8"/>
    <text class="cm-svg-label" x="560" y="252" text-anchor="middle">On losing</text>
    <text class="cm-svg-sub" x="560" y="272" text-anchor="middle">Pause(), then removeAllHosts()</text>
  </svg>
  <figcaption>A standby instance holds connections and goroutines but performs no work until it wins.</figcaption>
</figure>

The `won` callback resumes work and then writes to `discoverTrigger`, a channel of capacity one that
`hosts.Process` selects on alongside the interval ticker. A new leader therefore runs a discovery
immediately instead of waiting up to a full interval, so provisioning continues within seconds of a
failover.

The `lost` callback pauses and calls `removeAllHosts`, which empties the map and then drains the `work`
channel in a non-blocking loop until it is empty. A host already passed to a worker stays with that
worker, but the worker's `isCurrent` check rejects it before provisioning starts.
[Discovery and the work queue]({{% relref "discovery" %}}) covers that check.

Campaigning uses `backoff.TwentySec`, so a failover can take up to a minute of standby time before a
new leader starts working.

## Failure handling in the election path

`hosts.Process` launches `startElection` in a goroutine, so its returned error is discarded. If
`fw.NewElection` fails, the instance stays paused and logs nothing further.

`elect.Start` runs in a nested goroutine and calls `log.Fatalf` on error, ending the process rather
than leaving the instance paused:

```go
go func() {
	err := elect.Start(ctx)
	if err != nil {
		log.Fatalf("Leader election failed to start: %s", err)
	}
}()
```

`startElection` defers `wg.Done()` and returns once the election is registered, so its `WaitGroup`
entry clears at setup rather than at shutdown.

## Single-instance deployment

With `leader_election` unset, `paused` stays false for the life of the process and every gate passes.
A single instance needs no election, and the documentation recommends that deployment unless high
availability is required.
[Configuration File]({{% relref "/configuration/provisioner" %}}) covers the operational side.

{{% notice style="tip" title="Next" %}}
[Reference and map]({{% relref "reference" %}}) collects the file map, configuration keys, and
metrics in one place.
{{% /notice %}}
