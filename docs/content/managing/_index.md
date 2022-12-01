+++
title = "Fleet Management"
toc = true
weight = 30
pre = "<b>3. </b>"
+++

Choria Server that is enabled for Provisioning expose a number of endpoints that can be used to interact with it in ways that is not otherwise possible.

{{% notice style="warning" %}}
These features are almost all extremely dangerous and should be used with great caution.
{{% /notice %}}

These abilities are not possible without enabling Provisioning in the server, so they are off by default. You should ensure your site security policies have adequate control over these actions.

## Checking if upgrades are supported

{{% notice secondary "Version Hint" code-branch %}}
This requires Choria 0.27.0 and newer which is due to ship early 2023
{{% /notice %}}

If you are running a new enough CLI and Server you can check if a running server supports version upgrades:

```nohighlight
$ choria inventory 73ce620b5a21.choria.local
Inventory for 73ce620b5a21.choria.local

  Choria Server Statistics:

                    Version: 0.99.0.20221201 (upgradable during provisioning)
...
```

This will indicate if the JWT or custom build flags enable version upgrades and will be disabled by default, requiring the JWT was made with `--upgrade` option.

## Restarting Servers

A fleet that is capable of provisioning will allow you to restart them over Choria RPC using CLI or other Client:

```nohighlight
$ choria req choria_provision restart token=s3cret --batch 100 --batch-sleep 30
```
In this case the `token` is the same as supplied when creating the JWT in the `--token` argument. You can use any discovery filters and other options as shown here.

Servers will restart and continue as normal after a randomized sleep up to 10 seconds.

## Initiating Provisioning 

Once a fleet of nodes is deployed using Provisioner they will not again update their configuration unless instructed to or if they encounter a problem that might be fixed by provisioning again such as credentials expiring.

You can though initiate provisioning ad-hoc using the Choria CLI or other Client:

```nohighlight
$ choria req choria_provision reprovision token=s3cret --batch 100 --batch-sleep 30
```

In this case the `token` is the same as supplied when creating the JWT in the `--token` argument. You can use any discovery filters and other options as shown here.

Servers will restart into provisioning mode after a randomized sleep up to 10 seconds.

## Shutting down Servers

When provisioning is enabled the servers can be shut down remotely, this will trigger a exit code 0 shutdown that will not trigger a systemd restart.

This should be used with great caution, since, once shut down the machines will not return without manual intervention via reboot or service restart. Treat this like a Big Red Button to emergency shut down the service.

```nohighlight
$ choria req choria_provision shutdown token=s3cret --batch 100 --batch-sleep 30
```

In this case the `token` is the same as supplied when creating the JWT in the `--token` argument. You can use any discovery filters and other options as shown here.

Servers will shut down after a short delay.
