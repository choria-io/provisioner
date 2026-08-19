# Discovery and the work queue

Broadcast discovery and lifecycle events both reach the same admission function, `add`. Everything
after `add` is identical.

{{% notice style="note" title="Where it lives" %}}
`hosts`: fleet-level scheduling. Files: `hosts.go` for the main loop, discovery, and the `add`
and `remove` pair; `event.go` for the lifecycle listener; `provisioner.go` for the worker pool and the
finisher.
{{% /notice %}}

## Startup order

`hosts.Process` (`hosts/hosts.go:36`) runs for the life of the process. It stores the framework,
configuration, and logger in package variables, opens a connector, and publishes a `startup` lifecycle
event for the `provisioner` component so a fleet operator can see the instance start.

It then starts goroutines in a fixed order:

<ol class="cm-steps">
  <li><b>Leader election, when enabled</b> If <code>leader_election</code> is set, <code>conf.Pause()</code> runs first so the instance starts inert, then <code>startElection</code> begins campaigning.</li>
  <li><b>The lifecycle listener</b> One goroutine subscribed to startup events for the configured component.</li>
  <li><b>The finisher</b> One goroutine draining the <code>done</code> channel.</li>
  <li><b>The worker pool</b> <code>cfg.Workers</code> goroutines, each running <code>provisioner()</code>.</li>
</ol>

`hosts.Process` then sets every Prometheus counter and gauge to zero, so the full metric set appears
on `/metrics` before the first server arrives. It runs `discover(ctx)` once, then blocks on a `select`
over the interval ticker, the election trigger channel, and context cancellation.

<figure class="cm-diagram">
  <svg viewBox="0 0 760 280" role="img" aria-label="Two discovery sources feed the add function. Admitted hosts go onto a buffered work channel, then to worker goroutines, then onto a done channel, through the finisher, and out of the hosts map. The hosts map feeds back into add as the membership check.">
    <defs>
      <marker id="dsah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="dsahf" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-faint)"/></marker>
    </defs>
    <!-- sources -->
    <rect class="cm-svg-box" x="20" y="20" width="170" height="56" rx="8"/>
    <text class="cm-svg-label" x="105" y="44" text-anchor="middle">Interval ticker</text>
    <text class="cm-svg-sub" x="105" y="62" text-anchor="middle">broadcast discovery</text>
    <rect class="cm-svg-box" x="20" y="110" width="170" height="56" rx="8"/>
    <text class="cm-svg-label" x="105" y="134" text-anchor="middle">Startup events</text>
    <text class="cm-svg-sub" x="105" y="152" text-anchor="middle">lifecycle subscription</text>
    <!-- add -->
    <rect x="250" y="65" width="140" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="320" y="89" text-anchor="middle" style="fill:var(--cm-accent)">add()</text>
    <text class="cm-svg-sub" x="320" y="107" text-anchor="middle">dedup and admit</text>
    <!-- work -->
    <rect x="440" y="65" width="140" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent) 10%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="510" y="89" text-anchor="middle" style="fill:var(--cm-accent)">work chan</text>
    <text class="cm-svg-sub" x="510" y="107" text-anchor="middle">buffered 50000</text>
    <!-- workers -->
    <rect x="620" y="65" width="120" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="680" y="89" text-anchor="middle" style="fill:var(--cm-accent)">workers</text>
    <text class="cm-svg-sub" x="680" y="107" text-anchor="middle">N goroutines</text>
    <!-- done -->
    <rect x="620" y="190" width="120" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent) 10%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="680" y="214" text-anchor="middle" style="fill:var(--cm-accent)">done chan</text>
    <text class="cm-svg-sub" x="680" y="232" text-anchor="middle">buffered 50000</text>
    <!-- finisher -->
    <rect class="cm-svg-box" x="400" y="190" width="140" height="56" rx="8"/>
    <text class="cm-svg-label" x="470" y="214" text-anchor="middle">finisher</text>
    <text class="cm-svg-sub" x="470" y="232" text-anchor="middle">calls remove()</text>
    <!-- hosts map -->
    <rect x="180" y="190" width="140" height="56" rx="8" fill="color-mix(in srgb, var(--cm-accent2) 14%, transparent)" stroke="var(--cm-accent2)"/>
    <text class="cm-svg-label" x="250" y="214" text-anchor="middle" style="fill:var(--cm-accent2)">hosts map</text>
    <text class="cm-svg-sub" x="250" y="232" text-anchor="middle">identity to Host</text>
    <!-- edges in -->
    <line x1="190" y1="48" x2="246" y2="84" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <line x1="190" y1="138" x2="246" y2="102" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <line x1="390" y1="93" x2="436" y2="93" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <line x1="580" y1="93" x2="616" y2="93" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <!-- edges out -->
    <line x1="680" y1="121" x2="680" y2="186" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <text class="cm-svg-sub" x="600" y="158" text-anchor="end">isCurrent() drops stale entries</text>
    <line x1="616" y1="218" x2="544" y2="218" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <line x1="396" y1="218" x2="324" y2="218" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#dsah)"/>
    <!-- feedback -->
    <line x1="250" y1="186" x2="250" y2="125" stroke="var(--cm-faint)" stroke-width="1.5" stroke-dasharray="5 4" marker-end="url(#dsahf)"/>
    <text class="cm-svg-sub" x="196" y="160" text-anchor="middle">membership</text>
  </svg>
  <figcaption>Both sources admit through <code>add</code>. A worker checks the map, not the queue, before acting.</figcaption>
</figure>

## Broadcast discovery

`discoverProvisionableNodes` (`hosts/hosts.go:201`) builds a filter for the `choria_provision` agent
and runs a broadcast discovery against the `provisioning` collective:

```go
bd := broadcast.New(fw)
nodes, err := bd.Discover(ctx,
	broadcast.Collective("provisioning"),
	broadcast.Filter(f),
	broadcast.SlidingWindow(),
	broadcast.Timeout(2*time.Second))
```

The timeout is a fixed two seconds and the window slides, so the call returns as soon as replies stop
arriving rather than always waiting the full two seconds. Every returned identity becomes a new
`host.Host` and goes through `add`.

`discover` (`hosts/hosts.go:186`) checks the pause flag first. When paused it logs, returns before
sending the broadcast request, and leaves the cycle counter unchanged, so a standby instance only
campaigns for the election.

## Lifecycle startup events

`listen` (`hosts/event.go:26`) queue-subscribes to
`choria.lifecycle.event.startup.<lifecycle_component>`, where the component defaults to
`provision_mode_server`. `config.Load` rejects a component containing `.`, `>`, or `*`, because the
value is interpolated straight into a NATS subject and those characters would widen the subscription.

Each message goes through `handle`, which checks the pause flag, parses the event, and returns the
identity only for events of type `lifecycle.Startup`. `listen` therefore admits a server that has just
booted into provisioning mode without waiting for the next discovery cycle. Broadcast discovery and
the listener increment separate counters, `choria_provisioner_discovered` and
`choria_provisioner_event_discovered`, so their relative contribution is visible in monitoring.

## Admission and deduplication

`add` (`hosts/hosts.go:153`) takes the mutex, and no other function writes to the `work` channel.

For an identity already in the map, `add` compares the age of the new sighting against
`conf.IntervalDuration`:

```go
if time.Since(host.DiscoveredTime()) < conf.IntervalDuration {
	return false
}
removeUnlocked(host)
```

In that comparison, `host` is the newly constructed `Host`, whose `discovered` field `NewHost` sets to
`time.Now()`. The comparison therefore measures the age of the sighting being processed, not the age
of the entry already in the map.

{{% notice style="warning" title="Duplicates cost less than dropped nodes" %}}
A duplicate provision costs less than a node that never gets provisioned, so `add` re-admits on the
older sighting. A duplicate is self-limiting, because the second attempt times out on its first RPC
failure and `Provision` also refuses nodes that have been waiting too long. A dropped node stays
unconfigured until the next event or discovery cycle.
{{% /notice %}}

`add` then writes to the buffered `work` channel in a non-blocking `select`. If the buffer is full,
`add` logs a failure and deletes the identity from the map, so the map holds only entries that are on
the queue.

## Workers

Each worker (`hosts/provisioner.go:15`) blocks on the `work` channel. It then calls `isCurrent`, which
looks the identity up in the map, before calling `provisionTarget`:

```go
if !isCurrent(host) {
	continue
}
```

An entry can be on the queue but no longer in the map, which happens when `removeAllHosts` runs after
a lost election. The worker skips the entry and does not send it to `done`, because `removeAllHosts`
already deleted the identity.

The worker then calls `provisionTarget`, which increments `choria_provisioner_busy_workers` for the
duration and delegates to `host.Provision`. The return value carries a delay flag alongside the error:

| Outcome | What the worker does |
|---------|----------------------|
| Error | Increments `choria_provisioner_provision_errors`, logs, and sends to `done` immediately. |
| Success, `delay` false | Sends to `done` immediately, so the host can be rediscovered on the next cycle. |
| Success, `delay` true | Sends to `done` after 60 seconds via `time.AfterFunc`. |

The 60 second delay leaves the node in the map while it restarts, so the next discovery cycle skips
it. After an in-place version upgrade the flag is false instead, and the node is re-provisioned on
the next cycle.

## The finisher

One goroutine drains `done` and calls `remove`, which deletes the identity from the map and updates
both the `choria_provisioner_waiting_nodes` and `choria_provisioner_work_queue_entries` gauges. Only
the finisher calls `remove`, so workers never take the mutex for deletion and one goroutine writes
both gauges.

{{% notice style="tip" title="Next" %}}
[The provisioning cycle]({{% relref "provisioning-cycle" %}}) covers the RPC sequence a worker runs
against a server.
{{% /notice %}}
