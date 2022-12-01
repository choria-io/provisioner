+++
title = "Monitoring"
toc = true
weight = 40
pre = "<b>4. </b>"
+++

## Watching the Provisioning Process

Choria Servers publish `lifecycle` events, these can be viewed using `choria tool event --component provision_mode_server`.

Servers will publish `startup`, `shutdown` and `provisioned` events that will be shown in the rolling display. These lifecycle events are published in JSON mode on the network and could be stored in a Stream for analysis.

```nohighlight
$ choria tool event --component provision_mode_server
Waiting for events from topic choria.lifecycle.event.> on nats://broker.choria.local:4222
14:35:46 [startup] n1.choria.local: provision_mode_server version 0.26.15
14:36:30 [provisioned] n1.choria.local: provision_mode_server
14:36:31 [shutdown] n1.choria.local: provision_mode_server
```

## Viewing servers waiting for provisioning

One can view the active connections in the Choria Broker, these are nodes waiting to be provisioned.  Typically one would also see connections for the Provisioner in addition to these.

![Provisioning account connections](../prov-account-conns.png)

## Leader Election

One can determine the node that is currently the leader in the leader election:

```nohighlight
$ choria election view provisioner
Election bucket information for CHORIA_LEADER_ELECTION

       Created: 01 Jan 01 00:00 +0000
       Storage: File
  Maximum Time: 1m0s
      Replicas: 3 on hosts broker1*, broker2, broker3
     Elections: 7

╭───────────────────────────────────────────────╮
│               Active Elections                │
├─────────────┬─────────────────────────────────┤
│ Election    │ Leader                          │
├─────────────┼─────────────────────────────────┤
│ provisioner │ provisioner1.choria.local       │
╰─────────────┴─────────────────────────────────╯
```

A re-election can be forced using `choria election evict provisioner`.

## Prometheus Data

The Provisioner keep statistics in the Prometheus format, using this you can observe deployed counts, performance of individual RPC calls and more.

![AAA Dashboard](../provisioner-dashboard.png)

All the statics have a `site` label allowing you to easily aggregate a global cluster of AAA Services.

| Statistic                           | Descriptions                                                        |
|-------------------------------------|---------------------------------------------------------------------|
| choria_provisioner_rpc_time         | How long each RPC request takes                                     |
| choria_provisioner_helper_time      | How long the helper takes to run                                    |
| choria_provisioner_discovered       | How many nodes are discovered using the broadcast discovery         |
| choria_provisioner_event_discovered | How many nodes were discovered due to events being fired about them |
| choria_provisioner_discover_cycles  | How many discovery cycles were ran                                  |
| choria_provisioner_rpc_errors       | How many times a RPC request failed                                 |
| choria_provisioner_helper_errors    | How many times the helper failed to run                             |
| choria_provisioner_discovery_errors | How many times the discovery failed to run                          |
| choria_provisioner_provision_errors | How many times provisioning failed                                  |
| choria_provisioner_paused           | 1 when the backplane paused operations, 0 otherwise                 |
| choria_provisioner_busy_workers     | How many workers are busy processing servers                        |
| choria_provisioner_provisioned      | Host many nodes were successfully provisioned                       |

We have a published [Grafana Dashboard](https://grafana.com/grafana/dashboards/12431-choria-server-provisioner/) for this statistics.
