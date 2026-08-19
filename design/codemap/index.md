# Code Map

Choria Provisioner is a single Go binary, about 2,400 lines across four packages. It finds Choria
Servers that have no configuration, asks a site-supplied helper what their configuration should be,
writes it over Choria RPC, and restarts them.

{{% notice style="note" title="Snapshot" %}}
Generated 2026-08-19 against commit `aca2b62` on branch `main`, with a clean working tree.
{{% /notice %}}

## The provisioning flow

A freshly installed Choria Server with provisioning enabled connects to the broker's `provisioning`
collective and answers only the `choria_provision` agent. Provisioner is a client on that same
collective. It finds such a server through broadcast discovery or a lifecycle event and puts it on
the work queue. A worker then runs a fixed sequence of RPC calls against that one server: fetch its
JWT, fetch its key material, fetch its inventory, ask the helper for a configuration, send the
configuration back, and restart it. The helper handles certificate authorities, naming schemes, fact
sources, and policy decisions. It is a program in any language that reads JSON on standard input and
writes JSON on standard output.

<figure class="cm-diagram">
  <svg viewBox="0 0 760 300" role="img" aria-label="Unprovisioned Choria Servers connect to the broker's provisioning collective. Provisioner discovers them through the broker, calls a local helper program for each server's configuration, and pushes that configuration back over the broker.">
    <defs>
      <marker id="ovah" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent)"/></marker>
      <marker id="ovah2" markerWidth="9" markerHeight="9" refX="7" refY="3" orient="auto"><path d="M0,0 L7,3 L0,6 Z" fill="var(--cm-accent2)"/></marker>
    </defs>
    <!-- fleet -->
    <rect class="cm-svg-box" x="20" y="50" width="175" height="92" rx="8"/>
    <text class="cm-svg-label" x="107" y="84" text-anchor="middle">Choria Servers</text>
    <text class="cm-svg-sub" x="107" y="104" text-anchor="middle">provisioning mode</text>
    <text class="cm-svg-sub" x="107" y="120" text-anchor="middle">no configuration yet</text>
    <!-- broker -->
    <rect x="290" y="50" width="180" height="92" rx="8" fill="color-mix(in srgb, var(--cm-accent) 10%, transparent)" stroke="var(--cm-accent)"/>
    <text class="cm-svg-label" x="380" y="84" text-anchor="middle" style="fill:var(--cm-accent)">Choria Broker</text>
    <text class="cm-svg-sub" x="380" y="104" text-anchor="middle">provisioning collective</text>
    <text class="cm-svg-sub" x="380" y="120" text-anchor="middle">lifecycle events</text>
    <!-- provisioner -->
    <rect x="565" y="50" width="175" height="92" rx="8" fill="color-mix(in srgb, var(--cm-accent) 18%, transparent)" stroke="var(--cm-accent)" stroke-width="2"/>
    <text class="cm-svg-label" x="652" y="84" text-anchor="middle" style="fill:var(--cm-accent)">Provisioner</text>
    <text class="cm-svg-sub" x="652" y="104" text-anchor="middle">discover, queue, work</text>
    <text class="cm-svg-sub" x="652" y="120" text-anchor="middle">N worker goroutines</text>
    <!-- inbound lane -->
    <line x1="195" y1="76" x2="286" y2="76" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#ovah)"/>
    <text class="cm-svg-sub" x="240" y="68" text-anchor="middle">startup event</text>
    <line x1="470" y1="76" x2="561" y2="76" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#ovah)"/>
    <text class="cm-svg-sub" x="515" y="68" text-anchor="middle">discover</text>
    <!-- outbound lane -->
    <line x1="565" y1="122" x2="474" y2="122" stroke="var(--cm-accent2)" stroke-width="2" marker-end="url(#ovah2)"/>
    <text class="cm-svg-sub" x="519" y="140" text-anchor="middle">RPC</text>
    <line x1="290" y1="122" x2="199" y2="122" stroke="var(--cm-accent2)" stroke-width="2" marker-end="url(#ovah2)"/>
    <text class="cm-svg-sub" x="244" y="140" text-anchor="middle">configure, restart</text>
    <!-- helper -->
    <rect class="cm-svg-box" x="565" y="210" width="175" height="66" rx="8"/>
    <text class="cm-svg-label" x="652" y="240" text-anchor="middle">Helper</text>
    <text class="cm-svg-sub" x="652" y="258" text-anchor="middle">any language, JSON I/O</text>
    <line x1="620" y1="142" x2="620" y2="206" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#ovah)"/>
    <line x1="685" y1="210" x2="685" y2="146" stroke="var(--cm-accent)" stroke-width="2" marker-end="url(#ovah)"/>
    <text class="cm-svg-sub" x="530" y="180" text-anchor="middle">host JSON out</text>
    <text class="cm-svg-sub" x="530" y="196" text-anchor="middle">config JSON back</text>
    <!-- outside world -->
    <rect class="cm-svg-box" x="20" y="210" width="175" height="66" rx="8" stroke-dasharray="4 3"/>
    <text class="cm-svg-label" x="107" y="240" text-anchor="middle">CA or Issuer</text>
    <text class="cm-svg-sub" x="107" y="258" text-anchor="middle">reached by the helper</text>
    <line x1="565" y1="243" x2="199" y2="243" stroke="var(--cm-faint)" stroke-width="1.5" stroke-dasharray="5 4" marker-end="url(#ovah)"/>
  </svg>
  <figcaption>Discovery and events both reach the same worker pool. Environment-specific work happens in the helper.</figcaption>
</figure>

## A startup log

A Provisioner that wins the election logs its discovery interval and worker count, followed by one
line for each host admitted and one for each RPC issued:

<div class="cm-terminal">
  <div class="cm-tbar"><i class="r"></i><i class="y"></i><i class="g"></i></div>
  <div class="cm-tbody"><span class="c-dim">INFO</span>  Choria Provisioner starting using configuration file /etc/choria-provisioner/choria-provisioner.yaml. Discovery interval 1m using 4 workers
<span class="c-dim">INFO</span>  Starting leader election against 'provisioner'
<span class="c-warn">WARN</span>  Became leader after winning election
<span class="c-dim">INFO</span>  Triggered a discovery after becoming leader
<span class="c-dim">INFO</span>  Looking for provisionable hosts
<span class="c-accent">INFO</span>  Adding n1.example.net to the provision list after discovering it
<span class="c-accent">INFO</span>  Adding n2.example.net to the provision list after receiving an event
<span class="c-tool">INFO</span>  Provisioning n1.example.net
<span class="c-tool">INFO</span>  Fetching JWT   <span class="c-dim">-&gt; Fetching Inventory -&gt; Configuring node -&gt; Restarting node</span>
<span class="c-ok">INFO</span>  Provisioned n1.example.net</div>
</div>

## Pages in this map

{{< subpages >}}

{{% notice style="tip" title="Next" %}}
[Architecture]({{% relref "architecture" %}}) covers the package layering.
[Discovery and the work queue]({{% relref "discovery" %}}) follows a server from first sighting to a
worker.
{{% /notice %}}
